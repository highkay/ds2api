import { useCallback, useEffect, useMemo, useState } from 'react'

import {
    fetchSettings,
    getExportData,
    postImportData,
    postPassword,
    putSettings,
} from './settingsApi'

const MAX_AUTO_FETCH_FAILURES = 3

const DEFAULT_RUNTIME = {
    account_max_inflight: 1,
    account_max_queue: 0,
    global_max_inflight: 1,
    token_refresh_interval_hours: 6,
    upstream_max_attempts: 1,
    retry_after_muted: false,
    retry_after_http_429: false,
    retry_after_http_403: false,
    retry_after_network: false,
    retry_after_http_5xx: false,
    allow_cooldown_account_fallback: false,
    risk_breaker_enabled: true,
    risk_breaker_window_seconds: 600,
    risk_breaker_mute_cooldown_seconds: 3600,
    risk_breaker_hard_mute_count: 2,
    risk_breaker_hard_cooldown_seconds: 21600,
    risk_breaker_http_429_threshold: 5,
    risk_breaker_http_403_threshold: 2,
    risk_breaker_soft_cooldown_seconds: 900,
    caller_max_inflight: 2,
    max_prompt_chars: 60000,
    max_ref_files_per_request: 8,
    max_inline_files_per_request: 4,
    prompt_risk_guard_enabled: true,
    prompt_block_rules: [],
    allow_auto_delete_all: false,
    disable_upstream_file_uploads: false,
    account_health_enabled: true,
    account_health_recovery_window_seconds: 900,
    account_health_max_cooldown_seconds: 21600,
    account_health_cooldown_429_seconds: 900,
    account_health_cooldown_403_seconds: 3600,
    account_health_cooldown_auth_seconds: 3600,
    account_health_cooldown_5xx_seconds: 120,
    account_health_cooldown_network_seconds: 30,
    account_health_cooldown_empty_seconds: 300,
    account_health_cooldown_muted_seconds: 3600,
}

const DEFAULT_FORM = {
    admin: { jwt_expire_hours: 24 },
    runtime: DEFAULT_RUNTIME,
    compat: { strip_reference_markers: true },
    responses: { store_ttl_seconds: 900 },
    embeddings: { provider: '' },
    auto_delete: { mode: 'none' },
    history_split: { enabled: true, trigger_after_turns: 1 },
    model_aliases_text: '{}',
}

function parseJSONMap(raw, fieldName, t) {
    const text = String(raw || '').trim()
    if (!text) {
        return {}
    }
    let parsed
    try {
        parsed = JSON.parse(text)
    } catch (_e) {
        throw new Error(t('settings.invalidJsonField', { field: fieldName }))
    }
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
        throw new Error(t('settings.invalidJsonField', { field: fieldName }))
    }
    return parsed
}

function normalizeAutoDeleteMode(raw) {
    const mode = String(raw?.mode || '').trim().toLowerCase()
    if (mode === 'none' || mode === 'single' || mode === 'all') {
        return mode
    }
    if (Boolean(raw?.sessions)) {
        return 'all'
    }
    return 'none'
}

function runtimeNumber(runtime, key) {
    return Number(runtime?.[key] ?? DEFAULT_RUNTIME[key])
}

function runtimeBool(runtime, key) {
    return Boolean(runtime?.[key] ?? DEFAULT_RUNTIME[key])
}

function runtimeRules(runtime, key) {
    return Array.isArray(runtime?.[key]) ? runtime[key] : DEFAULT_RUNTIME[key]
}

function fromServerForm(data) {
    const runtime = data.runtime || {}
    return {
        admin: { jwt_expire_hours: Number(data.admin?.jwt_expire_hours ?? 24) },
        runtime: {
            account_max_inflight: runtimeNumber(runtime, 'account_max_inflight'),
            account_max_queue: runtimeNumber(runtime, 'account_max_queue'),
            global_max_inflight: runtimeNumber(runtime, 'global_max_inflight'),
            token_refresh_interval_hours: runtimeNumber(runtime, 'token_refresh_interval_hours'),
            upstream_max_attempts: runtimeNumber(runtime, 'upstream_max_attempts'),
            retry_after_muted: runtimeBool(runtime, 'retry_after_muted'),
            retry_after_http_429: runtimeBool(runtime, 'retry_after_http_429'),
            retry_after_http_403: runtimeBool(runtime, 'retry_after_http_403'),
            retry_after_network: runtimeBool(runtime, 'retry_after_network'),
            retry_after_http_5xx: runtimeBool(runtime, 'retry_after_http_5xx'),
            allow_cooldown_account_fallback: runtimeBool(runtime, 'allow_cooldown_account_fallback'),
            risk_breaker_enabled: runtimeBool(runtime, 'risk_breaker_enabled'),
            risk_breaker_window_seconds: runtimeNumber(runtime, 'risk_breaker_window_seconds'),
            risk_breaker_mute_cooldown_seconds: runtimeNumber(runtime, 'risk_breaker_mute_cooldown_seconds'),
            risk_breaker_hard_mute_count: runtimeNumber(runtime, 'risk_breaker_hard_mute_count'),
            risk_breaker_hard_cooldown_seconds: runtimeNumber(runtime, 'risk_breaker_hard_cooldown_seconds'),
            risk_breaker_http_429_threshold: runtimeNumber(runtime, 'risk_breaker_http_429_threshold'),
            risk_breaker_http_403_threshold: runtimeNumber(runtime, 'risk_breaker_http_403_threshold'),
            risk_breaker_soft_cooldown_seconds: runtimeNumber(runtime, 'risk_breaker_soft_cooldown_seconds'),
            caller_max_inflight: runtimeNumber(runtime, 'caller_max_inflight'),
            max_prompt_chars: runtimeNumber(runtime, 'max_prompt_chars'),
            max_ref_files_per_request: runtimeNumber(runtime, 'max_ref_files_per_request'),
            max_inline_files_per_request: runtimeNumber(runtime, 'max_inline_files_per_request'),
            prompt_risk_guard_enabled: runtimeBool(runtime, 'prompt_risk_guard_enabled'),
            prompt_block_rules: runtimeRules(runtime, 'prompt_block_rules'),
            allow_auto_delete_all: runtimeBool(runtime, 'allow_auto_delete_all'),
            disable_upstream_file_uploads: runtimeBool(runtime, 'disable_upstream_file_uploads'),
            account_health_enabled: runtimeBool(runtime, 'account_health_enabled'),
            account_health_recovery_window_seconds: runtimeNumber(runtime, 'account_health_recovery_window_seconds'),
            account_health_max_cooldown_seconds: runtimeNumber(runtime, 'account_health_max_cooldown_seconds'),
            account_health_cooldown_429_seconds: runtimeNumber(runtime, 'account_health_cooldown_429_seconds'),
            account_health_cooldown_403_seconds: runtimeNumber(runtime, 'account_health_cooldown_403_seconds'),
            account_health_cooldown_auth_seconds: runtimeNumber(runtime, 'account_health_cooldown_auth_seconds'),
            account_health_cooldown_5xx_seconds: runtimeNumber(runtime, 'account_health_cooldown_5xx_seconds'),
            account_health_cooldown_network_seconds: runtimeNumber(runtime, 'account_health_cooldown_network_seconds'),
            account_health_cooldown_empty_seconds: runtimeNumber(runtime, 'account_health_cooldown_empty_seconds'),
            account_health_cooldown_muted_seconds: runtimeNumber(runtime, 'account_health_cooldown_muted_seconds'),
        },
        compat: {
            strip_reference_markers: data.compat?.strip_reference_markers ?? true,
        },
        responses: {
            store_ttl_seconds: Number(data.responses?.store_ttl_seconds ?? 900),
        },
        embeddings: {
            provider: data.embeddings?.provider || '',
        },
        auto_delete: {
            mode: normalizeAutoDeleteMode(data.auto_delete),
        },
        history_split: {
            enabled: true,
            trigger_after_turns: Number(data.history_split?.trigger_after_turns ?? 1),
        },
        model_aliases_text: JSON.stringify(data.model_aliases || {}, null, 2),
    }
}

function toServerPayload(form) {
    const runtime = form.runtime || DEFAULT_RUNTIME
    return {
        admin: { jwt_expire_hours: Number(form.admin.jwt_expire_hours) },
        runtime: {
            account_max_inflight: Number(runtime.account_max_inflight),
            account_max_queue: Number(runtime.account_max_queue),
            global_max_inflight: Number(runtime.global_max_inflight),
            token_refresh_interval_hours: Number(runtime.token_refresh_interval_hours),
            upstream_max_attempts: Number(runtime.upstream_max_attempts),
            retry_after_muted: Boolean(runtime.retry_after_muted),
            retry_after_http_429: Boolean(runtime.retry_after_http_429),
            retry_after_http_403: Boolean(runtime.retry_after_http_403),
            retry_after_network: Boolean(runtime.retry_after_network),
            retry_after_http_5xx: Boolean(runtime.retry_after_http_5xx),
            allow_cooldown_account_fallback: Boolean(runtime.allow_cooldown_account_fallback),
            risk_breaker_enabled: Boolean(runtime.risk_breaker_enabled),
            risk_breaker_window_seconds: Number(runtime.risk_breaker_window_seconds),
            risk_breaker_mute_cooldown_seconds: Number(runtime.risk_breaker_mute_cooldown_seconds),
            risk_breaker_hard_mute_count: Number(runtime.risk_breaker_hard_mute_count),
            risk_breaker_hard_cooldown_seconds: Number(runtime.risk_breaker_hard_cooldown_seconds),
            risk_breaker_http_429_threshold: Number(runtime.risk_breaker_http_429_threshold),
            risk_breaker_http_403_threshold: Number(runtime.risk_breaker_http_403_threshold),
            risk_breaker_soft_cooldown_seconds: Number(runtime.risk_breaker_soft_cooldown_seconds),
            caller_max_inflight: Number(runtime.caller_max_inflight),
            max_prompt_chars: Number(runtime.max_prompt_chars),
            max_ref_files_per_request: Number(runtime.max_ref_files_per_request),
            max_inline_files_per_request: Number(runtime.max_inline_files_per_request),
            prompt_risk_guard_enabled: Boolean(runtime.prompt_risk_guard_enabled),
            prompt_block_rules: Array.isArray(runtime.prompt_block_rules) ? runtime.prompt_block_rules : [],
            allow_auto_delete_all: Boolean(runtime.allow_auto_delete_all),
            disable_upstream_file_uploads: Boolean(runtime.disable_upstream_file_uploads),
            account_health_enabled: Boolean(runtime.account_health_enabled),
            account_health_recovery_window_seconds: Number(runtime.account_health_recovery_window_seconds),
            account_health_max_cooldown_seconds: Number(runtime.account_health_max_cooldown_seconds),
            account_health_cooldown_429_seconds: Number(runtime.account_health_cooldown_429_seconds),
            account_health_cooldown_403_seconds: Number(runtime.account_health_cooldown_403_seconds),
            account_health_cooldown_auth_seconds: Number(runtime.account_health_cooldown_auth_seconds),
            account_health_cooldown_5xx_seconds: Number(runtime.account_health_cooldown_5xx_seconds),
            account_health_cooldown_network_seconds: Number(runtime.account_health_cooldown_network_seconds),
            account_health_cooldown_empty_seconds: Number(runtime.account_health_cooldown_empty_seconds),
            account_health_cooldown_muted_seconds: Number(runtime.account_health_cooldown_muted_seconds),
        },
        compat: {
            strip_reference_markers: Boolean(form.compat?.strip_reference_markers ?? true),
        },
        responses: { store_ttl_seconds: Number(form.responses.store_ttl_seconds) },
        embeddings: { provider: String(form.embeddings.provider || '').trim() },
        auto_delete: { mode: normalizeAutoDeleteMode(form.auto_delete) },
        history_split: {
            enabled: true,
            trigger_after_turns: Number(form.history_split?.trigger_after_turns || 1),
        },
    }
}

export function useSettingsForm({ apiFetch, t, onMessage, onRefresh, onForceLogout, isVercel = false }) {
    const [loading, setLoading] = useState(false)
    const [saving, setSaving] = useState(false)
    const [changingPassword, setChangingPassword] = useState(false)
    const [importing, setImporting] = useState(false)
    const [exportData, setExportData] = useState(null)
    const [importMode, setImportMode] = useState('merge')
    const [importText, setImportText] = useState('')
    const [newPassword, setNewPassword] = useState('')
    const [consecutiveFailures, setConsecutiveFailures] = useState(0)
    const [autoFetchPaused, setAutoFetchPaused] = useState(false)
    const [lastError, setLastError] = useState('')
    const [settingsMeta, setSettingsMeta] = useState({
        default_password_warning: false,
        env_backed: false,
        needs_vercel_sync: false,
    })
    const [form, setForm] = useState(DEFAULT_FORM)

    const trackLoadFailure = useCallback(() => {
        setConsecutiveFailures((prev) => {
            const next = prev + 1
            if (isVercel && next >= MAX_AUTO_FETCH_FAILURES) {
                setAutoFetchPaused(true)
            }
            return next
        })
    }, [isVercel])

    const loadSettings = useCallback(async ({ manual = false } = {}) => {
        if (isVercel && autoFetchPaused && !manual) {
            return
        }
        setLoading(true)
        try {
            const { res, data } = await fetchSettings(apiFetch, t)
            if (!res.ok) {
                const detail = data.detail || t('settings.loadFailed')
                setLastError(detail)
                onMessage('error', detail)
                trackLoadFailure()
                return
            }
            setConsecutiveFailures(0)
            setAutoFetchPaused(false)
            setLastError('')
            setSettingsMeta({
                default_password_warning: Boolean(data.admin?.default_password_warning),
                env_backed: Boolean(data.env_backed),
                needs_vercel_sync: Boolean(data.needs_vercel_sync),
            })
            setForm(fromServerForm(data))
        } catch (e) {
            const detail = e?.message || t('settings.loadFailed')
            setLastError(detail)
            onMessage('error', detail)
            trackLoadFailure()
            // eslint-disable-next-line no-console
            console.error(e)
        } finally {
            setLoading(false)
        }
    }, [apiFetch, autoFetchPaused, isVercel, onMessage, t, trackLoadFailure])

    useEffect(() => {
        loadSettings()
    }, [loadSettings])

    const retryLoadSettings = useCallback(() => {
        setAutoFetchPaused(false)
        loadSettings({ manual: true })
    }, [loadSettings])

    const saveSettings = useCallback(async () => {
        let modelAliases = {}
        try {
            modelAliases = parseJSONMap(form.model_aliases_text, 'model_aliases', t)
        } catch (e) {
            onMessage('error', e.message)
            return
        }

        const payload = {
            ...toServerPayload(form),
            model_aliases: modelAliases,
        }

        setSaving(true)
        try {
            const { res, data } = await putSettings(apiFetch, payload)
            if (!res.ok) {
                onMessage('error', data.detail || t('settings.saveFailed'))
                return
            }
            onMessage('success', t('settings.saveSuccess'))
            if (typeof onRefresh === 'function') {
                onRefresh()
            }
            await loadSettings()
        } catch (e) {
            onMessage('error', t('settings.saveFailed'))
            // eslint-disable-next-line no-console
            console.error(e)
        } finally {
            setSaving(false)
        }
    }, [apiFetch, form, loadSettings, onMessage, onRefresh, t])

    const updatePassword = useCallback(async () => {
        if (String(newPassword || '').trim().length < 4) {
            onMessage('error', t('settings.passwordTooShort'))
            return
        }
        setChangingPassword(true)
        try {
            const { res, data } = await postPassword(apiFetch, newPassword.trim())
            if (!res.ok) {
                onMessage('error', data.detail || t('settings.passwordUpdateFailed'))
                return
            }
            onMessage('success', t('settings.passwordUpdated'))
            setNewPassword('')
            if (typeof onForceLogout === 'function') {
                onForceLogout()
            }
        } catch (_e) {
            onMessage('error', t('settings.passwordUpdateFailed'))
        } finally {
            setChangingPassword(false)
        }
    }, [apiFetch, newPassword, onForceLogout, onMessage, t])

    const loadExportData = useCallback(async () => {
        try {
            const { res, data } = await getExportData(apiFetch)
            if (!res.ok) {
                onMessage('error', data.detail || t('settings.exportFailed'))
                return null
            }
            setExportData(data)
            onMessage('success', t('settings.exportLoaded'))
            return data
        } catch (_e) {
            onMessage('error', t('settings.exportFailed'))
            return null
        }
    }, [apiFetch, onMessage, t])

    const downloadExportFile = useCallback(async () => {
        let latest = exportData
        if (!latest?.json) {
            const loaded = await loadExportData()
            if (!loaded) {
                return
            }
            latest = loaded
        }
        const jsonText = String(latest?.json || '').trim()
        if (!jsonText) {
            onMessage('error', t('settings.exportFailed'))
            return
        }
        const blob = new Blob([jsonText], { type: 'application/json;charset=utf-8' })
        const url = URL.createObjectURL(blob)
        const now = new Date()
        const pad = (n) => String(n).padStart(2, '0')
        const filename = `ds2api-config-backup-${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}.json`
        const link = document.createElement('a')
        link.href = url
        link.download = filename
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        URL.revokeObjectURL(url)
        onMessage('success', t('settings.exportDownloaded'))
    }, [exportData, loadExportData, onMessage, t])

    const loadImportFile = useCallback((file) => {
        if (!file) return
        const reader = new FileReader()
        reader.onload = () => {
            const text = String(reader.result || '')
            setImportText(text)
            onMessage('success', t('settings.importFileLoaded'))
        }
        reader.onerror = () => {
            onMessage('error', t('settings.importFileReadFailed'))
        }
        reader.readAsText(file, 'utf-8')
    }, [onMessage, t])

    const doImport = useCallback(async () => {
        if (!String(importText || '').trim()) {
            onMessage('error', t('settings.importEmpty'))
            return
        }
        let parsed
        try {
            parsed = JSON.parse(importText)
        } catch (_e) {
            onMessage('error', t('settings.importInvalidJson'))
            return
        }
        setImporting(true)
        try {
            const { res, data } = await postImportData(apiFetch, importMode, parsed)
            if (!res.ok) {
                onMessage('error', data.detail || t('settings.importFailed'))
                return
            }
            onMessage('success', t('settings.importSuccess', { mode: importMode }))
            if (typeof onRefresh === 'function') {
                onRefresh()
            }
            await loadSettings()
        } catch (_e) {
            onMessage('error', t('settings.importFailed'))
        } finally {
            setImporting(false)
        }
    }, [apiFetch, importMode, importText, loadSettings, onMessage, onRefresh, t])

    const syncHintVisible = useMemo(
        () => settingsMeta.env_backed || settingsMeta.needs_vercel_sync,
        [settingsMeta.env_backed, settingsMeta.needs_vercel_sync],
    )

    return {
        form,
        setForm,
        loading,
        saving,
        changingPassword,
        importing,
        exportData,
        importMode,
        setImportMode,
        importText,
        setImportText,
        newPassword,
        setNewPassword,
        consecutiveFailures,
        autoFetchPaused,
        lastError,
        settingsMeta,
        syncHintVisible,
        retryLoadSettings,
        saveSettings,
        updatePassword,
        loadExportData,
        downloadExportFile,
        loadImportFile,
        doImport,
    }
}
