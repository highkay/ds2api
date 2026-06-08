package settings

import (
	"ds2api/internal/account"
	"ds2api/internal/config"
)

func validateMergedRuntimeSettings(current config.RuntimeConfig, incoming *config.RuntimeConfig) error {
	return validateRuntimeSettings(mergeRuntimeConfig(current, incoming))
}

func mergeRuntimeConfig(current config.RuntimeConfig, incoming *config.RuntimeConfig) config.RuntimeConfig {
	merged := current
	if incoming != nil {
		if incoming.AccountMaxInflight > 0 {
			merged.AccountMaxInflight = incoming.AccountMaxInflight
		}
		if incoming.AccountMaxQueue >= 0 {
			merged.AccountMaxQueue = incoming.AccountMaxQueue
		}
		if incoming.GlobalMaxInflight > 0 {
			merged.GlobalMaxInflight = incoming.GlobalMaxInflight
		}
		if incoming.TokenRefreshIntervalHours > 0 {
			merged.TokenRefreshIntervalHours = incoming.TokenRefreshIntervalHours
		}
		if incoming.UpstreamMaxAttempts > 0 {
			merged.UpstreamMaxAttempts = incoming.UpstreamMaxAttempts
		}
		if incoming.RetryAfterMuted != nil {
			merged.RetryAfterMuted = incoming.RetryAfterMuted
		}
		if incoming.RetryAfterHTTP429 != nil {
			merged.RetryAfterHTTP429 = incoming.RetryAfterHTTP429
		}
		if incoming.RetryAfterHTTP403 != nil {
			merged.RetryAfterHTTP403 = incoming.RetryAfterHTTP403
		}
		if incoming.RetryAfterNetwork != nil {
			merged.RetryAfterNetwork = incoming.RetryAfterNetwork
		}
		if incoming.RetryAfterHTTP5xx != nil {
			merged.RetryAfterHTTP5xx = incoming.RetryAfterHTTP5xx
		}
		if incoming.AllowCooldownAccountFallback != nil {
			merged.AllowCooldownAccountFallback = incoming.AllowCooldownAccountFallback
		}
		if incoming.RiskBreakerEnabled != nil {
			merged.RiskBreakerEnabled = incoming.RiskBreakerEnabled
		}
		if incoming.RiskBreakerWindowSeconds > 0 {
			merged.RiskBreakerWindowSeconds = incoming.RiskBreakerWindowSeconds
		}
		if incoming.RiskBreakerMuteCooldownSeconds > 0 {
			merged.RiskBreakerMuteCooldownSeconds = incoming.RiskBreakerMuteCooldownSeconds
		}
		if incoming.RiskBreakerHardMuteCount > 0 {
			merged.RiskBreakerHardMuteCount = incoming.RiskBreakerHardMuteCount
		}
		if incoming.RiskBreakerHardCooldownSeconds > 0 {
			merged.RiskBreakerHardCooldownSeconds = incoming.RiskBreakerHardCooldownSeconds
		}
		if incoming.RiskBreakerHTTP429Threshold > 0 {
			merged.RiskBreakerHTTP429Threshold = incoming.RiskBreakerHTTP429Threshold
		}
		if incoming.RiskBreakerHTTP403Threshold > 0 {
			merged.RiskBreakerHTTP403Threshold = incoming.RiskBreakerHTTP403Threshold
		}
		if incoming.RiskBreakerSoftCooldownSeconds > 0 {
			merged.RiskBreakerSoftCooldownSeconds = incoming.RiskBreakerSoftCooldownSeconds
		}
		if incoming.CallerMaxInflight > 0 {
			merged.CallerMaxInflight = incoming.CallerMaxInflight
		}
		if incoming.MaxPromptChars > 0 {
			merged.MaxPromptChars = incoming.MaxPromptChars
		}
		if incoming.MaxRefFilesPerRequest > 0 {
			merged.MaxRefFilesPerRequest = incoming.MaxRefFilesPerRequest
		}
		if incoming.MaxInlineFilesPerRequest > 0 {
			merged.MaxInlineFilesPerRequest = incoming.MaxInlineFilesPerRequest
		}
		if incoming.PromptRiskGuardEnabled != nil {
			merged.PromptRiskGuardEnabled = incoming.PromptRiskGuardEnabled
		}
		if incoming.PromptBlockRules != nil {
			merged.PromptBlockRules = incoming.PromptBlockRules
		}
		if incoming.AllowAutoDeleteAll != nil {
			merged.AllowAutoDeleteAll = incoming.AllowAutoDeleteAll
		}
		if incoming.DisableUpstreamFileUploads != nil {
			merged.DisableUpstreamFileUploads = incoming.DisableUpstreamFileUploads
		}
		if incoming.AccountHealthEnabled != nil {
			merged.AccountHealthEnabled = incoming.AccountHealthEnabled
		}
		if incoming.AccountHealthRecoveryWindowSeconds > 0 {
			merged.AccountHealthRecoveryWindowSeconds = incoming.AccountHealthRecoveryWindowSeconds
		}
		if incoming.AccountHealthMaxCooldownSeconds > 0 {
			merged.AccountHealthMaxCooldownSeconds = incoming.AccountHealthMaxCooldownSeconds
		}
		if incoming.AccountHealthCooldown429Seconds > 0 {
			merged.AccountHealthCooldown429Seconds = incoming.AccountHealthCooldown429Seconds
		}
		if incoming.AccountHealthCooldown403Seconds > 0 {
			merged.AccountHealthCooldown403Seconds = incoming.AccountHealthCooldown403Seconds
		}
		if incoming.AccountHealthCooldownAuthSeconds > 0 {
			merged.AccountHealthCooldownAuthSeconds = incoming.AccountHealthCooldownAuthSeconds
		}
		if incoming.AccountHealthCooldown5xxSeconds > 0 {
			merged.AccountHealthCooldown5xxSeconds = incoming.AccountHealthCooldown5xxSeconds
		}
		if incoming.AccountHealthCooldownNetworkSeconds > 0 {
			merged.AccountHealthCooldownNetworkSeconds = incoming.AccountHealthCooldownNetworkSeconds
		}
		if incoming.AccountHealthCooldownEmptySeconds >= 0 {
			merged.AccountHealthCooldownEmptySeconds = incoming.AccountHealthCooldownEmptySeconds
		}
		if incoming.AccountHealthCooldownMutedSeconds > 0 {
			merged.AccountHealthCooldownMutedSeconds = incoming.AccountHealthCooldownMutedSeconds
		}
	}
	return merged
}

func (h *Handler) applyRuntimeSettings() {
	if h == nil || h.Store == nil || h.Pool == nil {
		return
	}
	accountCount := len(h.Store.Accounts())
	maxPer := h.Store.RuntimeAccountMaxInflight()
	recommended := defaultRuntimeRecommended(accountCount, maxPer)
	maxQueue := h.Store.RuntimeAccountMaxQueue(recommended)
	global := h.Store.RuntimeGlobalMaxInflight(recommended)
	h.Pool.ApplyRuntimeLimits(maxPer, maxQueue, global)
	h.Pool.ApplyHealthConfig(account.LoadHealthConfigFromStore(h.Store))
	h.Pool.ApplyRiskConfig(account.LoadRiskConfigFromStore(h.Store))
	h.Pool.ApplyRuntimePolicy(h.Store.RuntimeAllowCooldownAccountFallback())
}

func defaultRuntimeRecommended(accountCount, maxPer int) int {
	if maxPer <= 0 {
		maxPer = 1
	}
	if accountCount <= 0 {
		return maxPer
	}
	return accountCount * maxPer
}
