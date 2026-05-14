package settings

import (
	"ds2api/internal/account"
	"ds2api/internal/config"
)

func validateMergedRuntimeSettings(current config.RuntimeConfig, incoming *config.RuntimeConfig) error {
	merged := current
	if incoming != nil {
		if incoming.AccountMaxInflight > 0 {
			merged.AccountMaxInflight = incoming.AccountMaxInflight
		}
		if incoming.AccountMaxQueue > 0 {
			merged.AccountMaxQueue = incoming.AccountMaxQueue
		}
		if incoming.GlobalMaxInflight > 0 {
			merged.GlobalMaxInflight = incoming.GlobalMaxInflight
		}
		if incoming.TokenRefreshIntervalHours > 0 {
			merged.TokenRefreshIntervalHours = incoming.TokenRefreshIntervalHours
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
	return validateRuntimeSettings(merged)
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
