package settings

import (
	"net/http"
	"strings"

	"ds2api/internal/account"
	authn "ds2api/internal/auth"
	"ds2api/internal/config"
)

func (h *Handler) getSettings(w http.ResponseWriter, _ *http.Request) {
	snap := h.Store.Snapshot()
	recommended := defaultRuntimeRecommended(len(snap.Accounts), h.Store.RuntimeAccountMaxInflight())
	needsSync := config.IsVercel() && snap.VercelSyncHash != "" && snap.VercelSyncHash != h.computeSyncHash()
	health := account.LoadHealthConfigFromStore(h.Store)
	risk := account.LoadRiskConfigFromStore(h.Store)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"admin": map[string]any{
			"has_password_hash":        strings.TrimSpace(snap.Admin.PasswordHash) != "",
			"jwt_expire_hours":         h.Store.AdminJWTExpireHours(),
			"jwt_valid_after_unix":     snap.Admin.JWTValidAfterUnix,
			"default_password_warning": authn.UsingDefaultAdminKey(h.Store),
		},
		"runtime": map[string]any{
			"account_max_inflight":                    h.Store.RuntimeAccountMaxInflight(),
			"account_max_queue":                       h.Store.RuntimeAccountMaxQueue(recommended),
			"global_max_inflight":                     h.Store.RuntimeGlobalMaxInflight(recommended),
			"token_refresh_interval_hours":            h.Store.RuntimeTokenRefreshIntervalHours(),
			"upstream_max_attempts":                   h.Store.RuntimeUpstreamMaxAttempts(),
			"retry_after_muted":                       h.Store.RuntimeRetryAfterMuted(),
			"retry_after_http_429":                    h.Store.RuntimeRetryAfterHTTP429(),
			"retry_after_http_403":                    h.Store.RuntimeRetryAfterHTTP403(),
			"retry_after_network":                     h.Store.RuntimeRetryAfterNetwork(),
			"retry_after_http_5xx":                    h.Store.RuntimeRetryAfterHTTP5xx(),
			"allow_cooldown_account_fallback":         h.Store.RuntimeAllowCooldownAccountFallback(),
			"risk_breaker_enabled":                    risk.Enabled,
			"risk_breaker_window_seconds":             risk.WindowSeconds,
			"risk_breaker_mute_cooldown_seconds":      risk.MuteCooldownSeconds,
			"risk_breaker_hard_mute_count":            risk.HardMuteCount,
			"risk_breaker_hard_cooldown_seconds":      risk.HardCooldownSeconds,
			"risk_breaker_http_429_threshold":         risk.HTTP429Threshold,
			"risk_breaker_http_403_threshold":         risk.HTTP403Threshold,
			"risk_breaker_soft_cooldown_seconds":      risk.SoftCooldownSeconds,
			"caller_max_inflight":                     h.Store.RuntimeCallerMaxInflight(),
			"max_prompt_chars":                        h.Store.RuntimeMaxPromptChars(),
			"max_ref_files_per_request":               h.Store.RuntimeMaxRefFilesPerRequest(),
			"max_inline_files_per_request":            h.Store.RuntimeMaxInlineFilesPerRequest(),
			"allow_auto_delete_all":                   h.Store.RuntimeAllowAutoDeleteAll(),
			"disable_upstream_file_uploads":           !h.Store.UpstreamFileUploadsEnabled(),
			"account_health_enabled":                  health.Enabled,
			"account_health_recovery_window_seconds":  health.RecoveryWindowSeconds,
			"account_health_max_cooldown_seconds":     health.MaxCooldownSeconds,
			"account_health_cooldown_429_seconds":     health.Cooldown429Seconds,
			"account_health_cooldown_403_seconds":     health.Cooldown403Seconds,
			"account_health_cooldown_auth_seconds":    health.CooldownAuthSeconds,
			"account_health_cooldown_5xx_seconds":     health.Cooldown5xxSeconds,
			"account_health_cooldown_network_seconds": health.CooldownNetworkSeconds,
			"account_health_cooldown_empty_seconds":   health.CooldownEmptySeconds,
			"account_health_cooldown_muted_seconds":   health.CooldownMutedSeconds,
		},
		"compat":      snap.Compat,
		"responses":   snap.Responses,
		"embeddings":  snap.Embeddings,
		"auto_delete": snap.AutoDelete,
		"history_split": map[string]any{
			"enabled":             h.Store.HistorySplitEnabled(),
			"trigger_after_turns": h.Store.HistorySplitTriggerAfterTurns(),
		},
		"model_aliases":     snap.ModelAliases,
		"env_backed":        h.Store.IsEnvBacked(),
		"needs_vercel_sync": needsSync,
	})
}
