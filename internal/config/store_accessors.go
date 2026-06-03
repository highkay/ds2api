package config

import (
	"os"
	"strconv"
	"strings"
)

func (s *Store) ModelAliases() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := DefaultModelAliases()
	for k, v := range s.cfg.ModelAliases {
		key := strings.TrimSpace(lower(k))
		val := strings.TrimSpace(lower(v))
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func (s *Store) CompatWideInputStrictOutput() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Compat.WideInputStrictOutput == nil {
		return true
	}
	return *s.cfg.Compat.WideInputStrictOutput
}

func (s *Store) CompatStripReferenceMarkers() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Compat.StripReferenceMarkers == nil {
		return true
	}
	return *s.cfg.Compat.StripReferenceMarkers
}

func (s *Store) ToolcallMode() string {
	return "feature_match"
}

func (s *Store) ToolcallEarlyEmitConfidence() string {
	return "high"
}

func (s *Store) ResponsesStoreTTLSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Responses.StoreTTLSeconds > 0 {
		return s.cfg.Responses.StoreTTLSeconds
	}
	return 900
}

func (s *Store) EmbeddingsProvider() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return strings.TrimSpace(s.cfg.Embeddings.Provider)
}

func (s *Store) AutoDeleteMode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mode := strings.ToLower(strings.TrimSpace(s.cfg.AutoDelete.Mode))
	switch mode {
	case "none", "single", "all":
		return mode
	}
	if s.cfg.AutoDelete.Sessions {
		return "all"
	}
	return "none"
}

func (s *Store) AdminPasswordHash() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return strings.TrimSpace(s.cfg.Admin.PasswordHash)
}

func (s *Store) AdminJWTExpireHours() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Admin.JWTExpireHours > 0 {
		return s.cfg.Admin.JWTExpireHours
	}
	if raw := strings.TrimSpace(os.Getenv("DS2API_JWT_EXPIRE_HOURS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return 24
}

func (s *Store) AdminJWTValidAfterUnix() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Admin.JWTValidAfterUnix
}

func (s *Store) RuntimeAccountMaxInflight() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.AccountMaxInflight > 0 {
		return s.cfg.Runtime.AccountMaxInflight
	}
	if raw := strings.TrimSpace(os.Getenv("DS2API_ACCOUNT_MAX_INFLIGHT")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return 1
}

func (s *Store) RuntimeAccountMaxQueue(_ int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.AccountMaxQueue > 0 {
		return s.cfg.Runtime.AccountMaxQueue
	}
	if raw := strings.TrimSpace(os.Getenv("DS2API_ACCOUNT_MAX_QUEUE")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			return n
		}
	}
	return 0
}

func (s *Store) RuntimeGlobalMaxInflight(defaultSize int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.GlobalMaxInflight > 0 {
		return s.cfg.Runtime.GlobalMaxInflight
	}
	if raw := strings.TrimSpace(os.Getenv("DS2API_GLOBAL_MAX_INFLIGHT")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	if defaultSize < 0 {
		return 0
	}
	return defaultSize
}

func (s *Store) RuntimeTokenRefreshIntervalHours() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.TokenRefreshIntervalHours > 0 {
		return s.cfg.Runtime.TokenRefreshIntervalHours
	}
	return 6
}

func (s *Store) RuntimeAccountMuteScanIntervalSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.AccountMuteScanIntervalSeconds > 0 {
		return s.cfg.Runtime.AccountMuteScanIntervalSeconds
	}
	return 43200
}

func (s *Store) UpstreamFileUploadsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Runtime.DisableUpstreamFileUploads == nil || !*s.cfg.Runtime.DisableUpstreamFileUploads
}

type upstreamFileUploadsEnabledReader interface {
	UpstreamFileUploadsEnabled() bool
}

func UpstreamFileUploadsEnabledFrom(reader any) bool {
	if reader == nil {
		return true
	}
	if r, ok := reader.(upstreamFileUploadsEnabledReader); ok {
		return r.UpstreamFileUploadsEnabled()
	}
	return true
}

func (s *Store) AccountHealthEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.AccountHealthEnabled == nil {
		return true
	}
	return *s.cfg.Runtime.AccountHealthEnabled
}

func (s *Store) AccountHealthRecoveryWindowSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Runtime.AccountHealthRecoveryWindowSeconds
}

func (s *Store) AccountHealthMaxCooldownSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Runtime.AccountHealthMaxCooldownSeconds
}

func (s *Store) AccountHealthCooldown429Seconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Runtime.AccountHealthCooldown429Seconds
}

func (s *Store) AccountHealthCooldown403Seconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Runtime.AccountHealthCooldown403Seconds
}

func (s *Store) AccountHealthCooldownAuthSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Runtime.AccountHealthCooldownAuthSeconds
}

func (s *Store) AccountHealthCooldown5xxSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Runtime.AccountHealthCooldown5xxSeconds
}

func (s *Store) AccountHealthCooldownNetworkSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Runtime.AccountHealthCooldownNetworkSeconds
}

func (s *Store) AccountHealthCooldownEmptySeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Runtime.AccountHealthCooldownEmptySeconds
}

func (s *Store) AccountHealthCooldownMutedSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Runtime.AccountHealthCooldownMutedSeconds
}

func (s *Store) AutoDeleteSessions() bool {
	return s.AutoDeleteMode() != "none"
}

func (s *Store) HistorySplitEnabled() bool {
	return true
}

func (s *Store) HistorySplitTriggerAfterTurns() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.HistorySplit.TriggerAfterTurns == nil || *s.cfg.HistorySplit.TriggerAfterTurns <= 0 {
		return 1
	}
	return *s.cfg.HistorySplit.TriggerAfterTurns
}
