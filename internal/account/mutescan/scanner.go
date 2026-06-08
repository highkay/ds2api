package mutescan

import (
	"context"
	"strings"
	"sync"
	"time"

	"ds2api/internal/account"
	"ds2api/internal/config"
	dsclient "ds2api/internal/deepseek/client"
)

const (
	DefaultInterval = 12 * time.Hour
	MinInterval     = 30 * time.Second
	maxConcurrency  = 4
)

type Store interface {
	Accounts() []config.Account
	MarkAccountMuted(identifier string, muteUntil float64) error
	ClearAccountMute(identifier string) error
}

type Checker interface {
	GetAccountMuteStatus(ctx context.Context, token string) (*dsclient.AccountMuteStatus, error)
}

type Penalizer interface {
	PenalizeHealth(accountID string, kind account.PenaltyKind)
}

type Summary struct {
	Total   int
	Checked int
	Muted   int
	Cleared int
	Failed  int
}

type Scanner struct {
	store     Store
	checker   Checker
	penalizer Penalizer
	interval  time.Duration
}

func New(store Store, checker Checker, penalizer Penalizer, interval time.Duration) *Scanner {
	if interval <= 0 {
		interval = DefaultInterval
	}
	if interval < MinInterval {
		interval = MinInterval
	}
	return &Scanner{store: store, checker: checker, penalizer: penalizer, interval: interval}
}

func (s *Scanner) Start(ctx context.Context) {
	if s == nil || s.store == nil || s.checker == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	go s.run(ctx)
}

func (s *Scanner) run(ctx context.Context) {
	s.RefreshNow(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RefreshNow(ctx)
		}
	}
}

func (s *Scanner) RefreshNow(ctx context.Context) Summary {
	if s == nil || s.store == nil || s.checker == nil {
		return Summary{}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	accounts := s.store.Accounts()
	summary := Summary{Total: len(accounts)}
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

scan:
	for _, acc := range accounts {
		if ctx.Err() != nil {
			break
		}
		acc := acc
		identifier := acc.Identifier()
		if identifier == "" || !acc.IsActive() || strings.TrimSpace(acc.Token) == "" {
			continue
		}
		select {
		case <-ctx.Done():
			break scan
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			status, err := s.checker.GetAccountMuteStatus(ctx, acc.Token)
			if err != nil {
				config.Logger.Warn("[account_mute_scan] current-user probe failed", "account", identifier, "error", err)
				mu.Lock()
				summary.Failed++
				mu.Unlock()
				return
			}

			mu.Lock()
			summary.Checked++
			mu.Unlock()

			if status != nil && status.Muted {
				if err := s.store.MarkAccountMuted(identifier, status.MuteUntil); err != nil {
					config.Logger.Warn("[account_mute_scan] failed to mark muted account", "account", identifier, "error", err)
					mu.Lock()
					summary.Failed++
					mu.Unlock()
					return
				}
				if s.penalizer != nil {
					// Mute scans observe account state out-of-band. They should
					// cool the individual account without tripping live-request
					// fleet risk breakers for historical mute state.
					s.penalizer.PenalizeHealth(identifier, account.PenaltyMuted)
				}
				mu.Lock()
				summary.Muted++
				mu.Unlock()
				return
			}

			if acc.Muted || acc.MuteUntil > 0 {
				if err := s.store.ClearAccountMute(identifier); err != nil {
					config.Logger.Warn("[account_mute_scan] failed to clear account mute", "account", identifier, "error", err)
					mu.Lock()
					summary.Failed++
					mu.Unlock()
					return
				}
				mu.Lock()
				summary.Cleared++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	if summary.Checked > 0 || summary.Muted > 0 || summary.Cleared > 0 || summary.Failed > 0 {
		config.Logger.Info("[account_mute_scan] completed", "total", summary.Total, "checked", summary.Checked, "muted", summary.Muted, "cleared", summary.Cleared, "failed", summary.Failed)
	}
	return summary
}
