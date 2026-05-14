package account

import (
	"math"
	"strings"
	"time"
)

type PenaltyKind string

const (
	PenaltyUnknown    PenaltyKind = ""
	PenaltyHTTP429    PenaltyKind = "http_429"
	PenaltyHTTP403    PenaltyKind = "http_403"
	PenaltyAuthFailed PenaltyKind = "auth_failed"
	PenaltyHTTP5xx    PenaltyKind = "http_5xx"
	PenaltyNetwork    PenaltyKind = "network"
	PenaltyEmpty      PenaltyKind = "empty_output"
	PenaltyMuted      PenaltyKind = "muted"
)

type HealthConfig struct {
	Enabled                bool
	RecoveryWindowSeconds  int
	MaxCooldownSeconds     int
	Cooldown429Seconds     int
	Cooldown403Seconds     int
	CooldownAuthSeconds    int
	Cooldown5xxSeconds     int
	CooldownNetworkSeconds int
	CooldownEmptySeconds   int
	CooldownMutedSeconds   int
}

func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		Enabled:                true,
		RecoveryWindowSeconds:  300,
		MaxCooldownSeconds:     1800,
		Cooldown429Seconds:     30,
		Cooldown403Seconds:     60,
		CooldownAuthSeconds:    120,
		Cooldown5xxSeconds:     10,
		CooldownNetworkSeconds: 5,
		CooldownEmptySeconds:   0,
		CooldownMutedSeconds:   300,
	}
}

type HealthConfigReader interface {
	AccountHealthEnabled() bool
	AccountHealthRecoveryWindowSeconds() int
	AccountHealthMaxCooldownSeconds() int
	AccountHealthCooldown429Seconds() int
	AccountHealthCooldown403Seconds() int
	AccountHealthCooldownAuthSeconds() int
	AccountHealthCooldown5xxSeconds() int
	AccountHealthCooldownNetworkSeconds() int
	AccountHealthCooldownEmptySeconds() int
	AccountHealthCooldownMutedSeconds() int
}

func LoadHealthConfigFromStore(r HealthConfigReader) HealthConfig {
	if r == nil {
		return DefaultHealthConfig()
	}
	def := DefaultHealthConfig()
	cfg := HealthConfig{Enabled: r.AccountHealthEnabled()}
	cfg.RecoveryWindowSeconds = pickPositive(r.AccountHealthRecoveryWindowSeconds(), def.RecoveryWindowSeconds)
	cfg.MaxCooldownSeconds = pickPositive(r.AccountHealthMaxCooldownSeconds(), def.MaxCooldownSeconds)
	cfg.Cooldown429Seconds = pickPositive(r.AccountHealthCooldown429Seconds(), def.Cooldown429Seconds)
	cfg.Cooldown403Seconds = pickPositive(r.AccountHealthCooldown403Seconds(), def.Cooldown403Seconds)
	cfg.CooldownAuthSeconds = pickPositive(r.AccountHealthCooldownAuthSeconds(), def.CooldownAuthSeconds)
	cfg.Cooldown5xxSeconds = pickPositive(r.AccountHealthCooldown5xxSeconds(), def.Cooldown5xxSeconds)
	cfg.CooldownNetworkSeconds = pickPositive(r.AccountHealthCooldownNetworkSeconds(), def.CooldownNetworkSeconds)
	cfg.CooldownMutedSeconds = pickPositive(r.AccountHealthCooldownMutedSeconds(), def.CooldownMutedSeconds)
	if r.AccountHealthCooldownEmptySeconds() >= 0 {
		cfg.CooldownEmptySeconds = r.AccountHealthCooldownEmptySeconds()
	} else {
		cfg.CooldownEmptySeconds = def.CooldownEmptySeconds
	}
	return cfg
}

func pickPositive(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

type healthConfig struct {
	enabled             bool
	recoveryWindow      time.Duration
	maxCooldown         time.Duration
	baseCooldown429     time.Duration
	baseCooldown403     time.Duration
	baseCooldownAuth    time.Duration
	baseCooldown5xx     time.Duration
	baseCooldownNetwork time.Duration
	baseCooldownEmpty   time.Duration
	baseCooldownMuted   time.Duration
}

func (c HealthConfig) toInternal() healthConfig {
	return healthConfig{
		enabled:             c.Enabled,
		recoveryWindow:      time.Duration(c.RecoveryWindowSeconds) * time.Second,
		maxCooldown:         time.Duration(c.MaxCooldownSeconds) * time.Second,
		baseCooldown429:     time.Duration(c.Cooldown429Seconds) * time.Second,
		baseCooldown403:     time.Duration(c.Cooldown403Seconds) * time.Second,
		baseCooldownAuth:    time.Duration(c.CooldownAuthSeconds) * time.Second,
		baseCooldown5xx:     time.Duration(c.Cooldown5xxSeconds) * time.Second,
		baseCooldownNetwork: time.Duration(c.CooldownNetworkSeconds) * time.Second,
		baseCooldownEmpty:   time.Duration(c.CooldownEmptySeconds) * time.Second,
		baseCooldownMuted:   time.Duration(c.CooldownMutedSeconds) * time.Second,
	}
}

const minWeight = 0.05
const weightTieEpsilon = 0.05

func weightDeltaByKind(kind PenaltyKind) float64 {
	switch kind {
	case PenaltyHTTP429:
		return 0.40
	case PenaltyHTTP403:
		return 0.60
	case PenaltyAuthFailed:
		return 0.70
	case PenaltyHTTP5xx:
		return 0.20
	case PenaltyNetwork:
		return 0.10
	case PenaltyEmpty:
		return 0.10
	case PenaltyMuted:
		return 0.80
	default:
		return 0.20
	}
}

func baseCooldownByKind(cfg healthConfig, kind PenaltyKind) time.Duration {
	switch kind {
	case PenaltyHTTP429:
		return cfg.baseCooldown429
	case PenaltyHTTP403:
		return cfg.baseCooldown403
	case PenaltyAuthFailed:
		return cfg.baseCooldownAuth
	case PenaltyHTTP5xx:
		return cfg.baseCooldown5xx
	case PenaltyNetwork:
		return cfg.baseCooldownNetwork
	case PenaltyEmpty:
		return cfg.baseCooldownEmpty
	case PenaltyMuted:
		return cfg.baseCooldownMuted
	default:
		return cfg.baseCooldown5xx
	}
}

type accountHealth struct {
	weight          float64
	failureCount    int
	lastFailureAt   time.Time
	lastFailureKind PenaltyKind
	lastSuccessAt   time.Time
	cooldownUntil   time.Time
}

func newAccountHealth() *accountHealth {
	return &accountHealth{weight: 1.0}
}

func (h *accountHealth) effectiveWeight(cfg healthConfig, now time.Time) float64 {
	if h == nil {
		return 1.0
	}
	w := h.weight
	if w < minWeight {
		w = minWeight
	}
	if w >= 1.0 {
		return 1.0
	}
	if h.lastFailureAt.IsZero() || cfg.recoveryWindow <= 0 {
		return w
	}
	elapsed := now.Sub(h.lastFailureAt)
	if elapsed <= 0 {
		return w
	}
	recovered := w + elapsed.Seconds()/cfg.recoveryWindow.Seconds()
	if recovered > 1.0 {
		return 1.0
	}
	return recovered
}

func (h *accountHealth) cooldownRemaining(now time.Time) time.Duration {
	if h == nil || h.cooldownUntil.IsZero() {
		return 0
	}
	if !h.cooldownUntil.After(now) {
		return 0
	}
	return h.cooldownUntil.Sub(now)
}

func (h *accountHealth) applyPenalty(cfg healthConfig, kind PenaltyKind, now time.Time) {
	h.failureCount++
	h.lastFailureAt = now
	h.lastFailureKind = kind
	h.weight -= weightDeltaByKind(kind)
	if h.weight < minWeight {
		h.weight = minWeight
	}
	base := baseCooldownByKind(cfg, kind)
	if base <= 0 {
		return
	}
	exp := math.Pow(2, float64(h.failureCount-1))
	if exp > 256 {
		exp = 256
	}
	cooldown := time.Duration(float64(base) * exp)
	if cfg.maxCooldown > 0 && cooldown > cfg.maxCooldown {
		cooldown = cfg.maxCooldown
	}
	h.cooldownUntil = now.Add(cooldown)
}

func (h *accountHealth) recordSuccess(now time.Time) {
	if h == nil {
		return
	}
	h.failureCount = 0
	h.lastSuccessAt = now
	h.cooldownUntil = time.Time{}
	h.lastFailureKind = ""
}

func ParsePenaltyKind(raw string) PenaltyKind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(PenaltyHTTP429):
		return PenaltyHTTP429
	case string(PenaltyHTTP403):
		return PenaltyHTTP403
	case string(PenaltyAuthFailed):
		return PenaltyAuthFailed
	case string(PenaltyHTTP5xx):
		return PenaltyHTTP5xx
	case string(PenaltyNetwork):
		return PenaltyNetwork
	case string(PenaltyEmpty):
		return PenaltyEmpty
	case string(PenaltyMuted):
		return PenaltyMuted
	}
	return PenaltyUnknown
}
