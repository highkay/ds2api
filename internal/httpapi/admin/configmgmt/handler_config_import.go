package configmgmt

import (
	"encoding/json"
	"net/http"
	"strings"

	"ds2api/internal/config"
)

func (h *Handler) configImport(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}

	mode := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("mode")))
	if mode == "" {
		mode = strings.TrimSpace(strings.ToLower(fieldString(req, "mode")))
	}
	if mode == "" {
		mode = "merge"
	}
	if mode != "merge" && mode != "replace" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "mode must be merge or replace"})
		return
	}

	payload := req
	if raw, ok := req["config"].(map[string]any); ok && len(raw) > 0 {
		payload = raw
	}
	rawJSON, err := json.Marshal(payload)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid config payload"})
		return
	}
	var incoming config.Config
	if err := json.Unmarshal(rawJSON, &incoming); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	incoming.ClearAccountTokens()

	importedKeys, importedAccounts := 0, 0
	err = h.Store.Update(func(c *config.Config) error {
		next := c.Clone()
		if mode == "replace" {
			next = incoming.Clone()
			next.Accounts = normalizeAndDedupeAccounts(next.Accounts)
			next.VercelSyncHash = c.VercelSyncHash
			next.VercelSyncTime = c.VercelSyncTime
			importedKeys = len(next.APIKeys)
			importedAccounts = len(next.Accounts)
		} else {
			var changed int
			next.APIKeys, changed = mergeAPIKeysPreferStructured(next.APIKeys, incoming.APIKeys)
			importedKeys += changed

			existingAccounts := map[string]struct{}{}
			for _, acc := range next.Accounts {
				acc = normalizeAccountForStorage(acc)
				key := accountDedupeKey(acc)
				if key != "" {
					existingAccounts[key] = struct{}{}
				}
			}
			for _, acc := range incoming.Accounts {
				acc = normalizeAccountForStorage(acc)
				key := accountDedupeKey(acc)
				if key == "" {
					continue
				}
				if _, ok := existingAccounts[key]; ok {
					continue
				}
				existingAccounts[key] = struct{}{}
				next.Accounts = append(next.Accounts, acc)
				importedAccounts++
			}

			if len(incoming.ModelAliases) > 0 {
				if next.ModelAliases == nil {
					next.ModelAliases = map[string]string{}
				}
				for k, v := range incoming.ModelAliases {
					next.ModelAliases[k] = v
				}
			}
			if incoming.Responses.StoreTTLSeconds > 0 {
				next.Responses.StoreTTLSeconds = incoming.Responses.StoreTTLSeconds
			}
			if strings.TrimSpace(incoming.Embeddings.Provider) != "" {
				next.Embeddings.Provider = incoming.Embeddings.Provider
			}
			if strings.TrimSpace(incoming.Admin.PasswordHash) != "" {
				next.Admin.PasswordHash = incoming.Admin.PasswordHash
			}
			if incoming.Admin.JWTExpireHours > 0 {
				next.Admin.JWTExpireHours = incoming.Admin.JWTExpireHours
			}
			if incoming.Admin.JWTValidAfterUnix > 0 {
				next.Admin.JWTValidAfterUnix = incoming.Admin.JWTValidAfterUnix
			}
			mergeRuntimePayload(&next.Runtime, incoming.Runtime, payload)
		}

		normalizeSettingsConfig(&next)
		if err := validateSettingsConfig(next); err != nil {
			return newRequestError(err.Error())
		}

		*c = next
		return nil
	})
	if err != nil {
		if detail, ok := requestErrorDetail(err); ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": detail})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}

	h.Pool.Reset()
	writeJSON(w, http.StatusOK, map[string]any{
		"success":           true,
		"mode":              mode,
		"imported_keys":     importedKeys,
		"imported_accounts": importedAccounts,
		"message":           "config imported",
	})
}

func configPayloadFieldPresent(payload map[string]any, section, name string) bool {
	raw, ok := payload[section].(map[string]any)
	if !ok {
		return false
	}
	_, exists := raw[name]
	return exists
}

func mergeRuntimePayload(next *config.RuntimeConfig, incoming config.RuntimeConfig, payload map[string]any) {
	if next == nil {
		return
	}
	if incoming.AccountMaxInflight > 0 {
		next.AccountMaxInflight = incoming.AccountMaxInflight
	}
	if runtimePayloadFieldPresent(payload, "account_max_queue") {
		next.AccountMaxQueue = incoming.AccountMaxQueue
	}
	if incoming.GlobalMaxInflight > 0 {
		next.GlobalMaxInflight = incoming.GlobalMaxInflight
	}
	if incoming.TokenRefreshIntervalHours > 0 {
		next.TokenRefreshIntervalHours = incoming.TokenRefreshIntervalHours
	}
	if incoming.AccountMuteScanIntervalSeconds > 0 {
		next.AccountMuteScanIntervalSeconds = incoming.AccountMuteScanIntervalSeconds
	}
	if incoming.UpstreamMaxAttempts > 0 {
		next.UpstreamMaxAttempts = incoming.UpstreamMaxAttempts
	}
	copyRuntimeBoolPtrIfPresent(&next.RetryAfterMuted, incoming.RetryAfterMuted, payload, "retry_after_muted")
	copyRuntimeBoolPtrIfPresent(&next.RetryAfterHTTP429, incoming.RetryAfterHTTP429, payload, "retry_after_http_429")
	copyRuntimeBoolPtrIfPresent(&next.RetryAfterHTTP403, incoming.RetryAfterHTTP403, payload, "retry_after_http_403")
	copyRuntimeBoolPtrIfPresent(&next.RetryAfterNetwork, incoming.RetryAfterNetwork, payload, "retry_after_network")
	copyRuntimeBoolPtrIfPresent(&next.RetryAfterHTTP5xx, incoming.RetryAfterHTTP5xx, payload, "retry_after_http_5xx")
	copyRuntimeBoolPtrIfPresent(&next.AllowCooldownAccountFallback, incoming.AllowCooldownAccountFallback, payload, "allow_cooldown_account_fallback")
	copyRuntimeBoolPtrIfPresent(&next.RiskBreakerEnabled, incoming.RiskBreakerEnabled, payload, "risk_breaker_enabled")
	if incoming.RiskBreakerWindowSeconds > 0 {
		next.RiskBreakerWindowSeconds = incoming.RiskBreakerWindowSeconds
	}
	if incoming.RiskBreakerMuteCooldownSeconds > 0 {
		next.RiskBreakerMuteCooldownSeconds = incoming.RiskBreakerMuteCooldownSeconds
	}
	if incoming.RiskBreakerHardMuteCount > 0 {
		next.RiskBreakerHardMuteCount = incoming.RiskBreakerHardMuteCount
	}
	if incoming.RiskBreakerHardCooldownSeconds > 0 {
		next.RiskBreakerHardCooldownSeconds = incoming.RiskBreakerHardCooldownSeconds
	}
	if incoming.RiskBreakerHTTP429Threshold > 0 {
		next.RiskBreakerHTTP429Threshold = incoming.RiskBreakerHTTP429Threshold
	}
	if incoming.RiskBreakerHTTP403Threshold > 0 {
		next.RiskBreakerHTTP403Threshold = incoming.RiskBreakerHTTP403Threshold
	}
	if incoming.RiskBreakerSoftCooldownSeconds > 0 {
		next.RiskBreakerSoftCooldownSeconds = incoming.RiskBreakerSoftCooldownSeconds
	}
	if incoming.CallerMaxInflight > 0 {
		next.CallerMaxInflight = incoming.CallerMaxInflight
	}
	if incoming.MaxPromptChars > 0 {
		next.MaxPromptChars = incoming.MaxPromptChars
	}
	if incoming.MaxRefFilesPerRequest > 0 {
		next.MaxRefFilesPerRequest = incoming.MaxRefFilesPerRequest
	}
	if incoming.MaxInlineFilesPerRequest > 0 {
		next.MaxInlineFilesPerRequest = incoming.MaxInlineFilesPerRequest
	}
	copyRuntimeBoolPtrIfPresent(&next.PromptRiskGuardEnabled, incoming.PromptRiskGuardEnabled, payload, "prompt_risk_guard_enabled")
	if runtimePayloadFieldPresent(payload, "prompt_block_rules") {
		next.PromptBlockRules = incoming.PromptBlockRules
	}
	copyRuntimeBoolPtrIfPresent(&next.AllowAutoDeleteAll, incoming.AllowAutoDeleteAll, payload, "allow_auto_delete_all")
	copyRuntimeBoolPtrIfPresent(&next.DisableUpstreamFileUploads, incoming.DisableUpstreamFileUploads, payload, "disable_upstream_file_uploads")
	copyRuntimeBoolPtrIfPresent(&next.AccountHealthEnabled, incoming.AccountHealthEnabled, payload, "account_health_enabled")
	if incoming.AccountHealthRecoveryWindowSeconds > 0 {
		next.AccountHealthRecoveryWindowSeconds = incoming.AccountHealthRecoveryWindowSeconds
	}
	if incoming.AccountHealthMaxCooldownSeconds > 0 {
		next.AccountHealthMaxCooldownSeconds = incoming.AccountHealthMaxCooldownSeconds
	}
	if incoming.AccountHealthCooldown429Seconds > 0 {
		next.AccountHealthCooldown429Seconds = incoming.AccountHealthCooldown429Seconds
	}
	if incoming.AccountHealthCooldown403Seconds > 0 {
		next.AccountHealthCooldown403Seconds = incoming.AccountHealthCooldown403Seconds
	}
	if incoming.AccountHealthCooldownAuthSeconds > 0 {
		next.AccountHealthCooldownAuthSeconds = incoming.AccountHealthCooldownAuthSeconds
	}
	if incoming.AccountHealthCooldown5xxSeconds > 0 {
		next.AccountHealthCooldown5xxSeconds = incoming.AccountHealthCooldown5xxSeconds
	}
	if incoming.AccountHealthCooldownNetworkSeconds > 0 {
		next.AccountHealthCooldownNetworkSeconds = incoming.AccountHealthCooldownNetworkSeconds
	}
	if runtimePayloadFieldPresent(payload, "account_health_cooldown_empty_seconds") {
		next.AccountHealthCooldownEmptySeconds = incoming.AccountHealthCooldownEmptySeconds
	}
	if incoming.AccountHealthCooldownMutedSeconds > 0 {
		next.AccountHealthCooldownMutedSeconds = incoming.AccountHealthCooldownMutedSeconds
	}
}

func runtimePayloadFieldPresent(payload map[string]any, name string) bool {
	return configPayloadFieldPresent(payload, "runtime", name)
}

func copyRuntimeBoolPtrIfPresent(dest **bool, incoming *bool, payload map[string]any, name string) {
	if runtimePayloadFieldPresent(payload, name) {
		*dest = incoming
	}
}
