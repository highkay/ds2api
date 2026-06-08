const CORE_FIELDS = [
    ['accountMaxInflight', 'account_max_inflight', 1, 256],
    ['accountMaxQueue', 'account_max_queue', 0, 200000],
    ['globalMaxInflight', 'global_max_inflight', 1, 200000],
    ['tokenRefreshIntervalHours', 'token_refresh_interval_hours', 1, 720],
    ['upstreamMaxAttempts', 'upstream_max_attempts', 1, 5],
    ['callerMaxInflight', 'caller_max_inflight', 1, 1000],
]

const TOGGLE_FIELDS = [
    ['retryAfterMuted', 'retry_after_muted'],
    ['retryAfterHTTP429', 'retry_after_http_429'],
    ['retryAfterHTTP403', 'retry_after_http_403'],
    ['retryAfterNetwork', 'retry_after_network'],
    ['retryAfterHTTP5xx', 'retry_after_http_5xx'],
    ['allowCooldownAccountFallback', 'allow_cooldown_account_fallback'],
    ['riskBreakerEnabled', 'risk_breaker_enabled'],
    ['promptRiskGuardEnabled', 'prompt_risk_guard_enabled'],
    ['disableUpstreamFileUploads', 'disable_upstream_file_uploads'],
    ['allowAutoDeleteAll', 'allow_auto_delete_all'],
    ['accountHealthEnabled', 'account_health_enabled'],
]

const RISK_FIELDS = [
    ['riskBreakerWindowSeconds', 'risk_breaker_window_seconds', 30, 86400],
    ['riskBreakerMuteCooldownSeconds', 'risk_breaker_mute_cooldown_seconds', 1, 86400],
    ['riskBreakerHardMuteCount', 'risk_breaker_hard_mute_count', 1, 100],
    ['riskBreakerHardCooldownSeconds', 'risk_breaker_hard_cooldown_seconds', 1, 86400],
    ['riskBreakerHTTP429Threshold', 'risk_breaker_http_429_threshold', 1, 10000],
    ['riskBreakerHTTP403Threshold', 'risk_breaker_http_403_threshold', 1, 10000],
    ['riskBreakerSoftCooldownSeconds', 'risk_breaker_soft_cooldown_seconds', 1, 86400],
]

const INPUT_FIELDS = [
    ['maxPromptChars', 'max_prompt_chars', 1000, 2000000],
    ['maxRefFilesPerRequest', 'max_ref_files_per_request', 1, 200],
    ['maxInlineFilesPerRequest', 'max_inline_files_per_request', 1, 200],
]

const HEALTH_FIELDS = [
    ['accountHealthRecoveryWindowSeconds', 'account_health_recovery_window_seconds', 1, 86400],
    ['accountHealthMaxCooldownSeconds', 'account_health_max_cooldown_seconds', 1, 86400],
    ['accountHealthCooldown429Seconds', 'account_health_cooldown_429_seconds', 1, 86400],
    ['accountHealthCooldown403Seconds', 'account_health_cooldown_403_seconds', 1, 86400],
    ['accountHealthCooldownAuthSeconds', 'account_health_cooldown_auth_seconds', 1, 86400],
    ['accountHealthCooldown5xxSeconds', 'account_health_cooldown_5xx_seconds', 1, 86400],
    ['accountHealthCooldownNetworkSeconds', 'account_health_cooldown_network_seconds', 1, 86400],
    ['accountHealthCooldownEmptySeconds', 'account_health_cooldown_empty_seconds', 0, 86400],
    ['accountHealthCooldownMutedSeconds', 'account_health_cooldown_muted_seconds', 1, 86400],
]

export default function RuntimeSection({ t, form, setForm }) {
    const runtime = form.runtime || {}
    const setRuntimeValue = (name, value) => {
        setForm((prev) => ({
            ...prev,
            runtime: { ...prev.runtime, [name]: value },
        }))
    }

    const numberField = ([labelKey, name, min, max]) => (
        <label key={name} className="text-sm space-y-2">
            <span className="text-muted-foreground">{t(`settings.${labelKey}`)}</span>
            <input
                type="number"
                min={min}
                max={max}
                step={1}
                value={runtime[name] ?? min}
                onChange={(e) => setRuntimeValue(name, Number(e.target.value || min))}
                className="w-full bg-background border border-border rounded-lg px-3 py-2"
            />
        </label>
    )

    const toggleField = ([labelKey, name]) => (
        <label key={name} className="flex items-center justify-between gap-3 border border-border rounded-lg px-3 py-2 text-sm">
            <span className="text-muted-foreground">{t(`settings.${labelKey}`)}</span>
            <input
                type="checkbox"
                checked={Boolean(runtime[name])}
                onChange={(e) => setRuntimeValue(name, e.target.checked)}
                className="h-4 w-4 rounded border-border"
            />
        </label>
    )

    return (
        <div className="bg-card border border-border rounded-lg p-5 space-y-5">
            <h3 className="font-semibold">{t('settings.runtimeTitle')}</h3>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {CORE_FIELDS.map(numberField)}
            </div>

            <div className="space-y-3">
                <h4 className="text-sm font-semibold">{t('settings.runtimeSafetyTitle')}</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {TOGGLE_FIELDS.map(toggleField)}
                </div>
            </div>

            <div className="space-y-3">
                <h4 className="text-sm font-semibold">{t('settings.riskBreakerTitle')}</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {RISK_FIELDS.map(numberField)}
                </div>
            </div>

            <div className="space-y-3">
                <h4 className="text-sm font-semibold">{t('settings.inputLimitTitle')}</h4>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    {INPUT_FIELDS.map(numberField)}
                </div>
            </div>

            <div className="space-y-3">
                <h4 className="text-sm font-semibold">{t('settings.accountHealthTitle')}</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {HEALTH_FIELDS.map(numberField)}
                </div>
            </div>
        </div>
    )
}
