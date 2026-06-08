package settings

import (
	"encoding/json"
	"fmt"
	"strings"

	"ds2api/internal/config"
)

func boolFrom(v any) bool {
	if v == nil {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.ToLower(strings.TrimSpace(x)) == "true"
	default:
		return false
	}
}

func setBoolPtrFrom(raw map[string]any, key string, dest **bool) {
	if v, exists := raw[key]; exists {
		b := boolFrom(v)
		*dest = &b
	}
}

func setRuntimeIntFrom(raw map[string]any, key string, min, max int, dest *int) error {
	v, exists := raw[key]
	if !exists {
		return nil
	}
	n := intFrom(v)
	if err := config.ValidateIntRange("runtime."+key, n, min, max, true); err != nil {
		return err
	}
	*dest = n
	return nil
}

func parseSettingsUpdateRequest(req map[string]any) (*config.AdminConfig, *config.RuntimeConfig, *config.CompatConfig, *config.ResponsesConfig, *config.EmbeddingsConfig, *config.AutoDeleteConfig, *config.HistorySplitConfig, map[string]string, error) {
	var (
		adminCfg        *config.AdminConfig
		runtimeCfg      *config.RuntimeConfig
		compatCfg       *config.CompatConfig
		respCfg         *config.ResponsesConfig
		embCfg          *config.EmbeddingsConfig
		autoDeleteCfg   *config.AutoDeleteConfig
		historySplitCfg *config.HistorySplitConfig
		aliasMap        map[string]string
	)

	if raw, ok := req["admin"].(map[string]any); ok {
		cfg := &config.AdminConfig{}
		if v, exists := raw["jwt_expire_hours"]; exists {
			n := intFrom(v)
			if err := config.ValidateIntRange("admin.jwt_expire_hours", n, 1, 720, true); err != nil {
				return nil, nil, nil, nil, nil, nil, nil, nil, err
			}
			cfg.JWTExpireHours = n
		}
		adminCfg = cfg
	}

	if raw, ok := req["runtime"].(map[string]any); ok {
		cfg := &config.RuntimeConfig{AccountMaxQueue: -1, AccountHealthCooldownEmptySeconds: -1}
		if v, exists := raw["account_max_inflight"]; exists {
			n := intFrom(v)
			if err := config.ValidateIntRange("runtime.account_max_inflight", n, 1, 256, true); err != nil {
				return nil, nil, nil, nil, nil, nil, nil, nil, err
			}
			cfg.AccountMaxInflight = n
		}
		if v, exists := raw["account_max_queue"]; exists {
			n := intFrom(v)
			if err := config.ValidateIntRange("runtime.account_max_queue", n, 0, 200000, true); err != nil {
				return nil, nil, nil, nil, nil, nil, nil, nil, err
			}
			cfg.AccountMaxQueue = n
		}
		if v, exists := raw["global_max_inflight"]; exists {
			n := intFrom(v)
			if err := config.ValidateIntRange("runtime.global_max_inflight", n, 1, 200000, true); err != nil {
				return nil, nil, nil, nil, nil, nil, nil, nil, err
			}
			cfg.GlobalMaxInflight = n
		}
		if v, exists := raw["token_refresh_interval_hours"]; exists {
			n := intFrom(v)
			if err := config.ValidateIntRange("runtime.token_refresh_interval_hours", n, 1, 720, true); err != nil {
				return nil, nil, nil, nil, nil, nil, nil, nil, err
			}
			cfg.TokenRefreshIntervalHours = n
		}
		if err := setRuntimeIntFrom(raw, "upstream_max_attempts", 1, 5, &cfg.UpstreamMaxAttempts); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		setBoolPtrFrom(raw, "retry_after_muted", &cfg.RetryAfterMuted)
		setBoolPtrFrom(raw, "retry_after_http_429", &cfg.RetryAfterHTTP429)
		setBoolPtrFrom(raw, "retry_after_http_403", &cfg.RetryAfterHTTP403)
		setBoolPtrFrom(raw, "retry_after_network", &cfg.RetryAfterNetwork)
		setBoolPtrFrom(raw, "retry_after_http_5xx", &cfg.RetryAfterHTTP5xx)
		setBoolPtrFrom(raw, "allow_cooldown_account_fallback", &cfg.AllowCooldownAccountFallback)
		setBoolPtrFrom(raw, "risk_breaker_enabled", &cfg.RiskBreakerEnabled)
		if err := setRuntimeIntFrom(raw, "risk_breaker_window_seconds", 30, 86400, &cfg.RiskBreakerWindowSeconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "risk_breaker_mute_cooldown_seconds", 1, 86400, &cfg.RiskBreakerMuteCooldownSeconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "risk_breaker_hard_mute_count", 1, 100, &cfg.RiskBreakerHardMuteCount); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "risk_breaker_hard_cooldown_seconds", 1, 86400, &cfg.RiskBreakerHardCooldownSeconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "risk_breaker_http_429_threshold", 1, 10000, &cfg.RiskBreakerHTTP429Threshold); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "risk_breaker_http_403_threshold", 1, 10000, &cfg.RiskBreakerHTTP403Threshold); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "risk_breaker_soft_cooldown_seconds", 1, 86400, &cfg.RiskBreakerSoftCooldownSeconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "caller_max_inflight", 1, 1000, &cfg.CallerMaxInflight); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "max_prompt_chars", 1000, 2000000, &cfg.MaxPromptChars); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "max_ref_files_per_request", 1, 200, &cfg.MaxRefFilesPerRequest); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "max_inline_files_per_request", 1, 200, &cfg.MaxInlineFilesPerRequest); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		setBoolPtrFrom(raw, "prompt_risk_guard_enabled", &cfg.PromptRiskGuardEnabled)
		if v, exists := raw["prompt_block_rules"]; exists {
			rules, err := promptBlockRulesFrom(v)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, nil, nil, err
			}
			cfg.PromptBlockRules = rules
		}
		setBoolPtrFrom(raw, "allow_auto_delete_all", &cfg.AllowAutoDeleteAll)
		setBoolPtrFrom(raw, "disable_upstream_file_uploads", &cfg.DisableUpstreamFileUploads)
		setBoolPtrFrom(raw, "account_health_enabled", &cfg.AccountHealthEnabled)
		if err := setRuntimeIntFrom(raw, "account_health_recovery_window_seconds", 1, 86400, &cfg.AccountHealthRecoveryWindowSeconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "account_health_max_cooldown_seconds", 1, 86400, &cfg.AccountHealthMaxCooldownSeconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "account_health_cooldown_429_seconds", 1, 86400, &cfg.AccountHealthCooldown429Seconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "account_health_cooldown_403_seconds", 1, 86400, &cfg.AccountHealthCooldown403Seconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "account_health_cooldown_auth_seconds", 1, 86400, &cfg.AccountHealthCooldownAuthSeconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "account_health_cooldown_5xx_seconds", 1, 86400, &cfg.AccountHealthCooldown5xxSeconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "account_health_cooldown_network_seconds", 1, 86400, &cfg.AccountHealthCooldownNetworkSeconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "account_health_cooldown_empty_seconds", 0, 86400, &cfg.AccountHealthCooldownEmptySeconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if err := setRuntimeIntFrom(raw, "account_health_cooldown_muted_seconds", 1, 86400, &cfg.AccountHealthCooldownMutedSeconds); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		if cfg.AccountMaxInflight > 0 && cfg.GlobalMaxInflight > 0 && cfg.GlobalMaxInflight < cfg.AccountMaxInflight {
			return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("runtime.global_max_inflight must be >= runtime.account_max_inflight")
		}
		runtimeCfg = cfg
	}

	if raw, ok := req["compat"].(map[string]any); ok {
		cfg := &config.CompatConfig{}
		if v, exists := raw["wide_input_strict_output"]; exists {
			b := boolFrom(v)
			cfg.WideInputStrictOutput = &b
		}
		if v, exists := raw["strip_reference_markers"]; exists {
			b := boolFrom(v)
			cfg.StripReferenceMarkers = &b
		}
		compatCfg = cfg
	}

	if raw, ok := req["responses"].(map[string]any); ok {
		cfg := &config.ResponsesConfig{}
		if v, exists := raw["store_ttl_seconds"]; exists {
			n := intFrom(v)
			if err := config.ValidateIntRange("responses.store_ttl_seconds", n, 30, 86400, true); err != nil {
				return nil, nil, nil, nil, nil, nil, nil, nil, err
			}
			cfg.StoreTTLSeconds = n
		}
		respCfg = cfg
	}

	if raw, ok := req["embeddings"].(map[string]any); ok {
		cfg := &config.EmbeddingsConfig{}
		if v, exists := raw["provider"]; exists {
			p := strings.TrimSpace(fmt.Sprintf("%v", v))
			if err := config.ValidateTrimmedString("embeddings.provider", p, false); err != nil {
				return nil, nil, nil, nil, nil, nil, nil, nil, err
			}
			cfg.Provider = p
		}
		embCfg = cfg
	}

	if raw, ok := req["model_aliases"].(map[string]any); ok {
		if aliasMap == nil {
			aliasMap = map[string]string{}
		}
		for k, v := range raw {
			key := strings.TrimSpace(k)
			val := strings.TrimSpace(fmt.Sprintf("%v", v))
			if key == "" || val == "" {
				continue
			}
			aliasMap[key] = val
		}
	}

	if raw, ok := req["auto_delete"].(map[string]any); ok {
		cfg := &config.AutoDeleteConfig{}
		if v, exists := raw["mode"]; exists {
			mode := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", v)))
			if err := config.ValidateAutoDeleteMode(mode); err != nil {
				return nil, nil, nil, nil, nil, nil, nil, nil, err
			}
			if mode == "" {
				mode = "none"
			}
			cfg.Mode = mode
		}
		if v, exists := raw["sessions"]; exists {
			cfg.Sessions = boolFrom(v)
		}
		autoDeleteCfg = cfg
	}

	if raw, ok := req["history_split"].(map[string]any); ok {
		cfg := &config.HistorySplitConfig{}
		enabled := true
		cfg.Enabled = &enabled
		if v, exists := raw["trigger_after_turns"]; exists {
			n := intFrom(v)
			if err := config.ValidateIntRange("history_split.trigger_after_turns", n, 1, 1000, true); err != nil {
				return nil, nil, nil, nil, nil, nil, nil, nil, err
			}
			cfg.TriggerAfterTurns = &n
		}
		if err := config.ValidateHistorySplitConfig(*cfg); err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		historySplitCfg = cfg
	}

	return adminCfg, runtimeCfg, compatCfg, respCfg, embCfg, autoDeleteCfg, historySplitCfg, aliasMap, nil
}

func promptBlockRulesFrom(v any) ([]config.PromptBlockRule, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("runtime.prompt_block_rules must be an array of objects")
	}
	var rules []config.PromptBlockRule
	if err := json.Unmarshal(b, &rules); err != nil {
		return nil, fmt.Errorf("runtime.prompt_block_rules must be an array of objects")
	}
	if rules == nil {
		rules = []config.PromptBlockRule{}
	}
	if err := config.ValidateRuntimeConfig(config.RuntimeConfig{PromptBlockRules: rules}); err != nil {
		return nil, err
	}
	return rules, nil
}
