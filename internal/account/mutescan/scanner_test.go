package mutescan

import (
	"context"
	"errors"
	"sync"
	"testing"

	"ds2api/internal/account"
	"ds2api/internal/config"
	dsclient "ds2api/internal/deepseek/client"
)

func TestScannerRefreshMarksClearsAndSkipsAccounts(t *testing.T) {
	store := &fakeMuteStore{accounts: []config.Account{
		{Email: "muted@example.com", Token: "tok-muted"},
		{Email: "old@example.com", Token: "tok-clear", Muted: true, MuteUntil: 9999},
		{Email: "err@example.com", Token: "tok-error", Muted: true, MuteUntil: 8888},
		{Email: "inactive@example.com", Token: "tok-inactive", Active: boolPtr(false)},
		{Email: "empty-token@example.com"},
		{Token: "tok-token-only"},
	}}
	checker := &fakeMuteChecker{
		statuses: map[string]*dsclient.AccountMuteStatus{
			"tok-muted": {Muted: true, MuteUntil: 12345},
			"tok-clear": {Muted: false},
		},
		errs: map[string]error{
			"tok-error": errors.New("probe failed"),
		},
	}
	penalizer := &fakeMutePenalizer{}
	scanner := New(store, checker, penalizer, 0)

	summary := scanner.RefreshNow(context.Background())
	if summary.Total != len(store.accounts) || summary.Checked != 2 || summary.Muted != 1 || summary.Cleared != 1 || summary.Failed != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if got := store.marked["muted@example.com"]; got != 12345 {
		t.Fatalf("expected muted@example.com marked until 12345, got %v", got)
	}
	if len(store.cleared) != 1 || store.cleared[0] != "old@example.com" {
		t.Fatalf("unexpected cleared accounts: %#v", store.cleared)
	}
	if len(penalizer.ids) != 1 || penalizer.ids[0] != "muted@example.com" {
		t.Fatalf("unexpected penalties: %#v", penalizer.ids)
	}
	if len(checker.tokens) != 3 {
		t.Fatalf("expected 3 probes, got %#v", checker.tokens)
	}
}

type fakeMuteStore struct {
	mu       sync.Mutex
	accounts []config.Account
	marked   map[string]float64
	cleared  []string
}

func (s *fakeMuteStore) Accounts() []config.Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]config.Account, len(s.accounts))
	copy(out, s.accounts)
	return out
}

func (s *fakeMuteStore) MarkAccountMuted(identifier string, muteUntil float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.marked == nil {
		s.marked = map[string]float64{}
	}
	s.marked[identifier] = muteUntil
	return nil
}

func (s *fakeMuteStore) ClearAccountMute(identifier string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleared = append(s.cleared, identifier)
	return nil
}

type fakeMuteChecker struct {
	mu       sync.Mutex
	statuses map[string]*dsclient.AccountMuteStatus
	errs     map[string]error
	tokens   []string
}

func (c *fakeMuteChecker) GetAccountMuteStatus(_ context.Context, token string) (*dsclient.AccountMuteStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokens = append(c.tokens, token)
	if err := c.errs[token]; err != nil {
		return nil, err
	}
	return c.statuses[token], nil
}

type fakeMutePenalizer struct {
	mu  sync.Mutex
	ids []string
}

func (p *fakeMutePenalizer) PenalizeHealth(accountID string, kind account.PenaltyKind) {
	if kind != account.PenaltyMuted {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ids = append(p.ids, accountID)
}

func boolPtr(v bool) *bool {
	return &v
}
