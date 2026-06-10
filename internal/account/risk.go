package account

import (
	"time"

	"ds2api/internal/config"
)

type RiskEventKind string

const (
	RiskEventMuted   RiskEventKind = "muted"
	RiskEventHTTP429 RiskEventKind = "http_429"
	RiskEventHTTP403 RiskEventKind = "http_403"
)

type RiskConfig struct {
	Enabled             bool
	WindowSeconds       int
	MuteCooldownSeconds int
	HardMuteCount       int
	HardCooldownSeconds int
	HTTP429Threshold    int
	HTTP403Threshold    int
	SoftCooldownSeconds int
}

func DefaultRiskConfig() RiskConfig {
	return RiskConfig{
		Enabled:             true,
		WindowSeconds:       7200,
		MuteCooldownSeconds: 3600,
		HardMuteCount:       2,
		HardCooldownSeconds: 21600,
		HTTP429Threshold:    5,
		HTTP403Threshold:    2,
		SoftCooldownSeconds: 900,
	}
}

type RiskConfigReader interface {
	RuntimeRiskBreakerEnabled() bool
	RuntimeRiskBreakerWindowSeconds() int
	RuntimeRiskBreakerMuteCooldownSeconds() int
	RuntimeRiskBreakerHardMuteCount() int
	RuntimeRiskBreakerHardCooldownSeconds() int
	RuntimeRiskBreakerHTTP429Threshold() int
	RuntimeRiskBreakerHTTP403Threshold() int
	RuntimeRiskBreakerSoftCooldownSeconds() int
}

func LoadRiskConfigFromStore(r RiskConfigReader) RiskConfig {
	if r == nil {
		return DefaultRiskConfig()
	}
	def := DefaultRiskConfig()
	return RiskConfig{
		Enabled:             r.RuntimeRiskBreakerEnabled(),
		WindowSeconds:       pickPositive(r.RuntimeRiskBreakerWindowSeconds(), def.WindowSeconds),
		MuteCooldownSeconds: pickPositive(r.RuntimeRiskBreakerMuteCooldownSeconds(), def.MuteCooldownSeconds),
		HardMuteCount:       pickPositive(r.RuntimeRiskBreakerHardMuteCount(), def.HardMuteCount),
		HardCooldownSeconds: pickPositive(r.RuntimeRiskBreakerHardCooldownSeconds(), def.HardCooldownSeconds),
		HTTP429Threshold:    pickPositive(r.RuntimeRiskBreakerHTTP429Threshold(), def.HTTP429Threshold),
		HTTP403Threshold:    pickPositive(r.RuntimeRiskBreakerHTTP403Threshold(), def.HTTP403Threshold),
		SoftCooldownSeconds: pickPositive(r.RuntimeRiskBreakerSoftCooldownSeconds(), def.SoftCooldownSeconds),
	}
}

type riskConfig struct {
	enabled          bool
	window           time.Duration
	muteCooldown     time.Duration
	hardMuteCount    int
	hardCooldown     time.Duration
	http429Threshold int
	http403Threshold int
	softCooldown     time.Duration
}

func (c RiskConfig) toInternal() riskConfig {
	return riskConfig{
		enabled:          c.Enabled,
		window:           time.Duration(c.WindowSeconds) * time.Second,
		muteCooldown:     time.Duration(c.MuteCooldownSeconds) * time.Second,
		hardMuteCount:    c.HardMuteCount,
		hardCooldown:     time.Duration(c.HardCooldownSeconds) * time.Second,
		http429Threshold: c.HTTP429Threshold,
		http403Threshold: c.HTTP403Threshold,
		softCooldown:     time.Duration(c.SoftCooldownSeconds) * time.Second,
	}
}

type riskEvent struct {
	kind      RiskEventKind
	at        time.Time
	accountID string
	callerID  string
	model     string
}

func RiskEventFromPenalty(kind PenaltyKind) (RiskEventKind, bool) {
	switch kind {
	case PenaltyMuted:
		return RiskEventMuted, true
	case PenaltyHTTP429:
		return RiskEventHTTP429, true
	case PenaltyHTTP403:
		return RiskEventHTTP403, true
	default:
		return "", false
	}
}

func (p *Pool) ApplyRiskConfig(cfg RiskConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.riskCfg = cfg.toInternal()
	p.notifyWaiterLocked()
}

func (p *Pool) ApplyRuntimePolicy(allowCooldownFallback bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.allowCooldownFallback = allowCooldownFallback
	p.notifyWaiterLocked()
}

func (p *Pool) RecordRiskEvent(kind RiskEventKind, accountID, callerID, model string) {
	if kind == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recordRiskEventLocked(kind, accountID, callerID, model, p.now())
}

func (p *Pool) recordRiskEventLocked(kind RiskEventKind, accountID, callerID, model string, now time.Time) {
	if !p.riskCfg.enabled {
		return
	}
	p.pruneRiskEventsLocked(now)
	p.riskEvents = append(p.riskEvents, riskEvent{
		kind:      kind,
		at:        now,
		accountID: accountID,
		callerID:  callerID,
		model:     model,
	})
	counts := p.riskCountsLocked()
	switch kind {
	case RiskEventMuted:
		reason := "muted"
		cooldown := p.riskCfg.muteCooldown
		if counts[RiskEventMuted] >= p.riskCfg.hardMuteCount {
			reason = "muted_hard"
			cooldown = p.riskCfg.hardCooldown
		}
		p.triggerRiskCooldownLocked(reason, cooldown, now)
	case RiskEventHTTP429:
		if counts[RiskEventHTTP429] >= p.riskCfg.http429Threshold {
			p.triggerRiskCooldownLocked("http_429_threshold", p.riskCfg.softCooldown, now)
		}
	case RiskEventHTTP403:
		if counts[RiskEventHTTP403] >= p.riskCfg.http403Threshold {
			p.triggerRiskCooldownLocked("http_403_threshold", p.riskCfg.softCooldown, now)
		}
	}
}

func (p *Pool) triggerRiskCooldownLocked(reason string, cooldown time.Duration, now time.Time) {
	if cooldown <= 0 {
		return
	}
	until := now.Add(cooldown)
	if !until.After(p.riskCooldownUntil) {
		return
	}
	p.riskCooldownUntil = until
	p.riskCooldownReason = reason
	p.drainWaitersLocked()
	config.Logger.Warn(
		"[account_risk_breaker] cooldown tripped",
		"reason", reason,
		"cooldown_seconds", int(cooldown.Seconds()),
		"until", until.Unix(),
	)
}

func (p *Pool) riskCoolingDownLocked(now time.Time) bool {
	return p.riskCfg.enabled && p.riskCooldownUntil.After(now)
}

func (p *Pool) pruneRiskEventsLocked(now time.Time) {
	if len(p.riskEvents) == 0 || p.riskCfg.window <= 0 {
		return
	}
	cutoff := now.Add(-p.riskCfg.window)
	keep := p.riskEvents[:0]
	for _, event := range p.riskEvents {
		if event.at.After(cutoff) || event.at.Equal(cutoff) {
			keep = append(keep, event)
		}
	}
	p.riskEvents = keep
}

func (p *Pool) riskCountsLocked() map[RiskEventKind]int {
	counts := map[RiskEventKind]int{}
	for _, event := range p.riskEvents {
		counts[event.kind]++
	}
	return counts
}

func (p *Pool) riskStatusLocked(now time.Time) map[string]any {
	p.pruneRiskEventsLocked(now)
	remaining := 0
	until := int64(0)
	if p.riskCoolingDownLocked(now) {
		remaining = int(p.riskCooldownUntil.Sub(now).Seconds())
		until = p.riskCooldownUntil.Unix()
	}
	counts := p.riskCountsLocked()
	return map[string]any{
		"enabled":            p.riskCfg.enabled,
		"cooling_down":       remaining > 0,
		"cooldown_remaining": remaining,
		"cooldown_until":     until,
		"reason":             p.riskCooldownReason,
		"window_seconds":     int(p.riskCfg.window.Seconds()),
		"muted_events":       counts[RiskEventMuted],
		"http_429_events":    counts[RiskEventHTTP429],
		"http_403_events":    counts[RiskEventHTTP403],
		"hard_mute_count":    p.riskCfg.hardMuteCount,
		"http_429_threshold": p.riskCfg.http429Threshold,
		"http_403_threshold": p.riskCfg.http403Threshold,
	}
}
