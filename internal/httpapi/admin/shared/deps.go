package shared

import (
	"context"
	"net/http"

	"ds2api/internal/account"
	"ds2api/internal/auth"
	"ds2api/internal/config"
	dsclient "ds2api/internal/deepseek/client"
)

type ConfigStore interface {
	Snapshot() config.Config
	Keys() []string
	Accounts() []config.Account
	FindAccount(identifier string) (config.Account, bool)
	UpdateAccountToken(identifier, token string) error
	UpdateAccountTestStatus(identifier, status string) error
	AccountTestStatus(identifier string) (string, bool)
	UpdateAccountRuntimeProbe(identifier string, probe config.AccountRuntimeProbe) error
	AccountRuntimeProbe(identifier string) (config.AccountRuntimeProbe, bool)
	Update(mutator func(*config.Config) error) error
	ExportJSONAndBase64() (string, string, error)
	IsEnvBacked() bool
	IsEnvWritebackEnabled() bool
	HasEnvConfigSource() bool
	ConfigPath() string
	SetVercelSync(hash string, ts int64) error
	AdminPasswordHash() string
	AdminJWTExpireHours() int
	AdminJWTValidAfterUnix() int64
	RuntimeAccountMaxInflight() int
	RuntimeAccountMaxQueue(defaultSize int) int
	RuntimeGlobalMaxInflight(defaultSize int) int
	RuntimeTokenRefreshIntervalHours() int
	RuntimeUpstreamMaxAttempts() int
	RuntimeRetryAfterMuted() bool
	RuntimeRetryAfterHTTP429() bool
	RuntimeRetryAfterHTTP403() bool
	RuntimeRetryAfterNetwork() bool
	RuntimeRetryAfterHTTP5xx() bool
	RuntimeAllowCooldownAccountFallback() bool
	RuntimeRiskBreakerEnabled() bool
	RuntimeRiskBreakerWindowSeconds() int
	RuntimeRiskBreakerMuteCooldownSeconds() int
	RuntimeRiskBreakerHardMuteCount() int
	RuntimeRiskBreakerHardCooldownSeconds() int
	RuntimeRiskBreakerHTTP429Threshold() int
	RuntimeRiskBreakerHTTP403Threshold() int
	RuntimeRiskBreakerSoftCooldownSeconds() int
	RuntimeCallerMaxInflight() int
	RuntimeMaxPromptChars() int
	RuntimeMaxRefFilesPerRequest() int
	RuntimeMaxInlineFilesPerRequest() int
	RuntimePromptRiskGuardEnabled() bool
	RuntimePromptBlockRules() []config.PromptBlockRule
	RuntimeAllowAutoDeleteAll() bool
	UpstreamFileUploadsEnabled() bool
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
	AutoDeleteMode() string
	HistorySplitEnabled() bool
	HistorySplitTriggerAfterTurns() int
	CompatStripReferenceMarkers() bool
	AutoDeleteSessions() bool
}

type PoolController interface {
	Reset()
	Status() map[string]any
	ApplyRuntimeLimits(maxInflightPerAccount, maxQueueSize, globalMaxInflight int)
	ApplyHealthConfig(cfg account.HealthConfig)
	ApplyRiskConfig(cfg account.RiskConfig)
	ApplyRuntimePolicy(allowCooldownFallback bool)
}

type OpenAIChatCaller interface {
	ChatCompletions(w http.ResponseWriter, r *http.Request)
}

type DeepSeekCaller interface {
	Login(ctx context.Context, acc config.Account) (string, error)
	CreateSession(ctx context.Context, a *auth.RequestAuth, maxAttempts int) (string, error)
	GetPow(ctx context.Context, a *auth.RequestAuth, maxAttempts int) (string, error)
	CallCompletion(ctx context.Context, a *auth.RequestAuth, payload map[string]any, powResp string, maxAttempts int) (*http.Response, error)
	GetSessionCountForToken(ctx context.Context, token string) (*dsclient.SessionStats, error)
	DeleteAllSessionsForToken(ctx context.Context, token string) error
	ValidateToken(ctx context.Context, token string) (*dsclient.TokenValidationResult, error)
	GetAccountCapabilities(ctx context.Context, token string, accountID string) (*dsclient.AccountCapabilities, error)
}

var _ ConfigStore = (*config.Store)(nil)
var _ PoolController = (*account.Pool)(nil)
var _ DeepSeekCaller = (*dsclient.Client)(nil)
