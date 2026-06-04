package account

import (
	"testing"
	"time"

	"ds2api/internal/config"
)

func newRiskPoolForTest(t *testing.T, cfg string) *Pool {
	t.Helper()
	t.Setenv("DS2API_ACCOUNT_MAX_INFLIGHT", "")
	t.Setenv("DS2API_ACCOUNT_MAX_QUEUE", "")
	t.Setenv("DS2API_GLOBAL_MAX_INFLIGHT", "")
	t.Setenv("DS2API_CONFIG_JSON", cfg)
	return NewPool(config.LoadStore())
}

func TestPoolRiskBreakerCoolsDownAfterMute(t *testing.T) {
	pool := newRiskPoolForTest(t, `{
		"keys":["k1"],
		"accounts":[{"email":"acc1@example.com","token":"token1"}],
		"runtime":{
			"risk_breaker_enabled":true,
			"risk_breaker_window_seconds":600,
			"risk_breaker_mute_cooldown_seconds":60,
			"risk_breaker_hard_mute_count":2,
			"risk_breaker_hard_cooldown_seconds":3600,
			"risk_breaker_http_429_threshold":5,
			"risk_breaker_http_403_threshold":2,
			"risk_breaker_soft_cooldown_seconds":30
		}
	}`)
	now := time.Unix(1000, 0)
	pool.now = func() time.Time { return now }

	pool.RecordRiskEvent(RiskEventMuted, "acc1@example.com", "caller", "deepseek-v4-pro")

	if _, ok := pool.Acquire("", nil); ok {
		t.Fatal("expected acquire to fail while risk breaker is cooling down")
	}
	status := pool.Status()
	risk, ok := status["risk"].(map[string]any)
	if !ok {
		t.Fatalf("missing risk status: %#v", status["risk"])
	}
	if got, _ := risk["cooling_down"].(bool); !got {
		t.Fatalf("expected risk breaker cooling down, risk=%v", risk)
	}
	if got, _ := risk["reason"].(string); got != "muted" {
		t.Fatalf("reason=%q want=muted risk=%v", got, risk)
	}
	if got := intFromAny(risk["muted_events"]); got != 1 {
		t.Fatalf("muted_events=%d want=1 risk=%v", got, risk)
	}
	if got := intFromAny(status["available"]); got != 0 {
		t.Fatalf("available=%d want=0 status=%v", got, status)
	}

	now = now.Add(61 * time.Second)
	acc, ok := pool.Acquire("", nil)
	if !ok {
		t.Fatal("expected acquire to succeed after risk cooldown expires")
	}
	if got := acc.Identifier(); got != "acc1@example.com" {
		t.Fatalf("acquired %q want acc1@example.com", got)
	}
}

func TestPoolRiskBreakerHardMuteExtendsCooldown(t *testing.T) {
	pool := newRiskPoolForTest(t, `{
		"keys":["k1"],
		"accounts":[{"email":"acc1@example.com","token":"token1"}],
		"runtime":{
			"risk_breaker_enabled":true,
			"risk_breaker_window_seconds":600,
			"risk_breaker_mute_cooldown_seconds":60,
			"risk_breaker_hard_mute_count":2,
			"risk_breaker_hard_cooldown_seconds":3600,
			"risk_breaker_http_429_threshold":5,
			"risk_breaker_http_403_threshold":2,
			"risk_breaker_soft_cooldown_seconds":30
		}
	}`)
	now := time.Unix(1000, 0)
	pool.now = func() time.Time { return now }

	pool.RecordRiskEvent(RiskEventMuted, "acc1@example.com", "caller-a", "deepseek-v4-pro")
	now = now.Add(10 * time.Second)
	pool.RecordRiskEvent(RiskEventMuted, "acc1@example.com", "caller-b", "deepseek-v4-pro")

	risk, _ := pool.Status()["risk"].(map[string]any)
	if got, _ := risk["reason"].(string); got != "muted_hard" {
		t.Fatalf("reason=%q want=muted_hard risk=%v", got, risk)
	}
	if got := intFromAny(risk["cooldown_remaining"]); got < 3500 {
		t.Fatalf("cooldown_remaining=%d want hard cooldown risk=%v", got, risk)
	}
}

func TestPoolRiskBreakerHTTP429Threshold(t *testing.T) {
	pool := newRiskPoolForTest(t, `{
		"keys":["k1"],
		"accounts":[{"email":"acc1@example.com","token":"token1"}],
		"runtime":{
			"risk_breaker_enabled":true,
			"risk_breaker_window_seconds":600,
			"risk_breaker_mute_cooldown_seconds":60,
			"risk_breaker_hard_mute_count":2,
			"risk_breaker_hard_cooldown_seconds":3600,
			"risk_breaker_http_429_threshold":2,
			"risk_breaker_http_403_threshold":2,
			"risk_breaker_soft_cooldown_seconds":30
		}
	}`)
	now := time.Unix(1000, 0)
	pool.now = func() time.Time { return now }

	pool.RecordRiskEvent(RiskEventHTTP429, "acc1@example.com", "caller-a", "deepseek-v4-pro")
	if _, ok := pool.Acquire("", nil); !ok {
		t.Fatal("expected first 429 below threshold not to cool the pool")
	}
	pool.Release("acc1@example.com")

	pool.RecordRiskEvent(RiskEventHTTP429, "acc1@example.com", "caller-a", "deepseek-v4-pro")
	if _, ok := pool.Acquire("", nil); ok {
		t.Fatal("expected acquire to fail after 429 threshold is reached")
	}
	risk, _ := pool.Status()["risk"].(map[string]any)
	if got, _ := risk["reason"].(string); got != "http_429_threshold" {
		t.Fatalf("reason=%q want=http_429_threshold risk=%v", got, risk)
	}
}

func intFromAny(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}
