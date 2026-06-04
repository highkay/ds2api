package config

import (
	"os"
	"strconv"
	"strings"
)

func envIntInRange(name string, minValue, maxValue int) (int, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < minValue || n > maxValue {
		return 0, false
	}
	return n, true
}

func envBool(name string) (bool, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return false, false
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false
	}
	return v, true
}

func boolPtrDefault(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

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
	if n, ok := envIntInRange("DS2API_ACCOUNT_MAX_INFLIGHT", 1, 256); ok {
		return n
	}
	return 1
}

func (s *Store) RuntimeAccountMaxQueue(_ int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.AccountMaxQueue > 0 {
		return s.cfg.Runtime.AccountMaxQueue
	}
	if n, ok := envIntInRange("DS2API_ACCOUNT_MAX_QUEUE", 0, 200000); ok {
		return n
	}
	return 0
}

func (s *Store) RuntimeGlobalMaxInflight(defaultSize int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.GlobalMaxInflight > 0 {
		return s.cfg.Runtime.GlobalMaxInflight
	}
	if n, ok := envIntInRange("DS2API_GLOBAL_MAX_INFLIGHT", 1, 200000); ok {
		return n
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

func (s *Store) RuntimeUpstreamMaxAttempts() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.UpstreamMaxAttempts > 0 {
		return s.cfg.Runtime.UpstreamMaxAttempts
	}
	if n, ok := envIntInRange("DS2API_UPSTREAM_MAX_ATTEMPTS", 1, 5); ok {
		return n
	}
	return 1
}

func (s *Store) RuntimeRetryAfterMuted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := envBool("DS2API_RETRY_AFTER_MUTED"); ok {
		return v
	}
	return boolPtrDefault(s.cfg.Runtime.RetryAfterMuted, false)
}

func (s *Store) RuntimeRetryAfterHTTP429() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := envBool("DS2API_RETRY_AFTER_HTTP_429"); ok {
		return v
	}
	return boolPtrDefault(s.cfg.Runtime.RetryAfterHTTP429, false)
}

func (s *Store) RuntimeRetryAfterHTTP403() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := envBool("DS2API_RETRY_AFTER_HTTP_403"); ok {
		return v
	}
	return boolPtrDefault(s.cfg.Runtime.RetryAfterHTTP403, false)
}

func (s *Store) RuntimeRetryAfterNetwork() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := envBool("DS2API_RETRY_AFTER_NETWORK"); ok {
		return v
	}
	return boolPtrDefault(s.cfg.Runtime.RetryAfterNetwork, false)
}

func (s *Store) RuntimeRetryAfterHTTP5xx() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := envBool("DS2API_RETRY_AFTER_HTTP_5XX"); ok {
		return v
	}
	return boolPtrDefault(s.cfg.Runtime.RetryAfterHTTP5xx, false)
}

func (s *Store) RuntimeAllowCooldownAccountFallback() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := envBool("DS2API_ALLOW_COOLDOWN_ACCOUNT_FALLBACK"); ok {
		return v
	}
	return boolPtrDefault(s.cfg.Runtime.AllowCooldownAccountFallback, false)
}

func (s *Store) RuntimeRiskBreakerEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return boolPtrDefault(s.cfg.Runtime.RiskBreakerEnabled, true)
}

func (s *Store) RuntimeRiskBreakerWindowSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.RiskBreakerWindowSeconds > 0 {
		return s.cfg.Runtime.RiskBreakerWindowSeconds
	}
	return 600
}

func (s *Store) RuntimeRiskBreakerMuteCooldownSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.RiskBreakerMuteCooldownSeconds > 0 {
		return s.cfg.Runtime.RiskBreakerMuteCooldownSeconds
	}
	return 3600
}

func (s *Store) RuntimeRiskBreakerHardMuteCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.RiskBreakerHardMuteCount > 0 {
		return s.cfg.Runtime.RiskBreakerHardMuteCount
	}
	return 2
}

func (s *Store) RuntimeRiskBreakerHardCooldownSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.RiskBreakerHardCooldownSeconds > 0 {
		return s.cfg.Runtime.RiskBreakerHardCooldownSeconds
	}
	return 21600
}

func (s *Store) RuntimeRiskBreakerHTTP429Threshold() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.RiskBreakerHTTP429Threshold > 0 {
		return s.cfg.Runtime.RiskBreakerHTTP429Threshold
	}
	return 5
}

func (s *Store) RuntimeRiskBreakerHTTP403Threshold() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.RiskBreakerHTTP403Threshold > 0 {
		return s.cfg.Runtime.RiskBreakerHTTP403Threshold
	}
	return 2
}

func (s *Store) RuntimeRiskBreakerSoftCooldownSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.RiskBreakerSoftCooldownSeconds > 0 {
		return s.cfg.Runtime.RiskBreakerSoftCooldownSeconds
	}
	return 900
}

func (s *Store) RuntimeCallerMaxInflight() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.CallerMaxInflight > 0 {
		return s.cfg.Runtime.CallerMaxInflight
	}
	return 2
}

func (s *Store) RuntimeMaxPromptChars() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.MaxPromptChars > 0 {
		return s.cfg.Runtime.MaxPromptChars
	}
	return 60000
}

func (s *Store) RuntimeMaxRefFilesPerRequest() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.MaxRefFilesPerRequest > 0 {
		return s.cfg.Runtime.MaxRefFilesPerRequest
	}
	return 8
}

func (s *Store) RuntimeMaxInlineFilesPerRequest() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg.Runtime.MaxInlineFilesPerRequest > 0 {
		return s.cfg.Runtime.MaxInlineFilesPerRequest
	}
	return 4
}

func (s *Store) RuntimeAllowAutoDeleteAll() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return boolPtrDefault(s.cfg.Runtime.AllowAutoDeleteAll, false)
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

type runtimeUpstreamMaxAttemptsReader interface {
	RuntimeUpstreamMaxAttempts() int
}

func RuntimeUpstreamMaxAttemptsFrom(reader any) int {
	if r, ok := reader.(runtimeUpstreamMaxAttemptsReader); ok {
		if n := r.RuntimeUpstreamMaxAttempts(); n > 0 {
			return n
		}
	}
	return 1
}

type runtimeMaxPromptCharsReader interface {
	RuntimeMaxPromptChars() int
}

func RuntimeMaxPromptCharsFrom(reader any) int {
	if r, ok := reader.(runtimeMaxPromptCharsReader); ok {
		if n := r.RuntimeMaxPromptChars(); n > 0 {
			return n
		}
	}
	return 60000
}

type runtimeMaxRefFilesPerRequestReader interface {
	RuntimeMaxRefFilesPerRequest() int
}

func RuntimeMaxRefFilesPerRequestFrom(reader any) int {
	if r, ok := reader.(runtimeMaxRefFilesPerRequestReader); ok {
		if n := r.RuntimeMaxRefFilesPerRequest(); n > 0 {
			return n
		}
	}
	return 8
}

type runtimeMaxInlineFilesPerRequestReader interface {
	RuntimeMaxInlineFilesPerRequest() int
}

func RuntimeMaxInlineFilesPerRequestFrom(reader any) int {
	if r, ok := reader.(runtimeMaxInlineFilesPerRequestReader); ok {
		if n := r.RuntimeMaxInlineFilesPerRequest(); n > 0 {
			return n
		}
	}
	return 4
}

type runtimeAllowAutoDeleteAllReader interface {
	RuntimeAllowAutoDeleteAll() bool
}

func RuntimeAllowAutoDeleteAllFrom(reader any) bool {
	if r, ok := reader.(runtimeAllowAutoDeleteAllReader); ok {
		return r.RuntimeAllowAutoDeleteAll()
	}
	return false
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
