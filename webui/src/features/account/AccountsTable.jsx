import { useState } from 'react'
import { AlertTriangle, ChevronLeft, ChevronRight, Check, Copy, Eye, Loader2, Pencil, Play, Plus, ShieldCheck, Timer, Trash2, FolderX } from 'lucide-react'
import clsx from 'clsx'

function normalizeID(value) {
    return String(value || '').trim()
}

function queueAccountID(item) {
    return normalizeID(item?.id || item?.account || item?.identifier || item?.email || item?.mobile)
}

function numberValue(value) {
    const n = Number(value)
    return Number.isFinite(n) ? n : 0
}

export default function AccountsTable({
    t,
    accounts,
    loadingAccounts,
    testing,
    testingAll,
    batchProgress,
    sessionCounts,
    deletingSessions,
    updatingProxy,
    totalAccounts,
    page,
    pageSize,
    totalPages,
    resolveAccountIdentifier,
    proxies,
    onTestAll,
    onShowAddAccount,
    onEditAccount,
    onTestAccount,
    onDeleteAccount,
    onDeleteAllSessions,
    onUpdateAccountProxy,
    onPrevPage,
    onNextPage,
    onPageSizeChange,
    searchQuery,
    onSearchChange,
    envBacked = false,
    queueStatus,
}) {
    const [copiedId, setCopiedId] = useState(null)

    const copyId = (id) => {
        navigator.clipboard.writeText(id).then(() => {
            setCopiedId(id)
            setTimeout(() => setCopiedId(null), 1500)
        })
    }

    const queueHealthByID = new Map(
        (queueStatus?.accounts || [])
            .map(item => [queueAccountID(item), item])
            .filter(([key]) => key)
    )

    return (
        <div className="bg-card border border-border rounded-xl overflow-hidden shadow-sm">
            <div className="p-6 border-b border-border flex flex-col md:flex-row md:items-center justify-between gap-4">
                <div>
                    <h2 className="text-lg font-semibold">{t('accountManager.accountsTitle')}</h2>
                    <p className="text-sm text-muted-foreground">{t('accountManager.accountsDesc')}</p>
                </div>
                <div className="flex flex-wrap gap-2">
                    <input
                        type="text"
                        value={searchQuery}
                        onChange={e => onSearchChange(e.target.value)}
                        placeholder={t('accountManager.searchPlaceholder')}
                        className="px-3 py-1.5 text-sm bg-muted border border-border rounded-lg focus:outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
                    />
                    <button
                        onClick={onTestAll}
                        disabled={testingAll || totalAccounts === 0}
                        className="flex items-center px-3 py-2 bg-secondary text-secondary-foreground rounded-lg hover:bg-secondary/80 transition-colors text-xs font-medium border border-border disabled:opacity-50"
                    >
                        {testingAll ? <span className="animate-spin mr-2">⟳</span> : <Play className="w-3 h-3 mr-2" />}
                        {t('accountManager.testAll')}
                    </button>
                    <button
                        onClick={onShowAddAccount}
                        className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors font-medium text-sm shadow-sm"
                    >
                        <Plus className="w-4 h-4" />
                        {t('accountManager.addAccount')}
                    </button>
                </div>
            </div>

            {testingAll && batchProgress.total > 0 && (
                <div className="p-4 border-b border-border bg-muted/30">
                    <div className="flex items-center justify-between text-sm mb-2">
                        <span className="font-medium">{t('accountManager.testingAllAccounts')}</span>
                        <span className="text-muted-foreground">{batchProgress.current} / {batchProgress.total}</span>
                    </div>
                    <div className="w-full bg-muted rounded-full h-2 overflow-hidden mb-4">
                        <div
                            className="bg-primary h-full transition-all duration-300"
                            style={{ width: `${(batchProgress.current / batchProgress.total) * 100}%` }}
                        />
                    </div>
                    {batchProgress.results.length > 0 && (
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-2 max-h-32 overflow-y-auto custom-scrollbar">
                            {batchProgress.results.map((r, i) => (
                                <div key={i} className={clsx(
                                    "text-xs px-2 py-1 rounded border truncate",
                                    r.success ? "bg-emerald-500/10 border-emerald-500/20 text-emerald-500" : "bg-destructive/10 border-destructive/20 text-destructive"
                                )}>
                                    {r.success ? '✓' : '✗'} {r.id}
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}

            <div className="divide-y divide-border">
                {loadingAccounts ? (
                    <div className="p-8 text-center text-muted-foreground">{t('actions.loading')}</div>
                ) : accounts.length > 0 ? (
                    accounts.map((acc, i) => {
                        const id = resolveAccountIdentifier(acc)
                        const assignedProxy = proxies.find(proxy => proxy.id === acc.proxy_id)
                        const runtimeUnknown = envBacked && !acc.test_status
                        const isActive = acc.test_status === 'ok' || acc.has_token
                        const health = queueHealthByID.get(id) || {}
                        const capabilities = acc.capabilities || acc.runtime_probe?.capabilities || {}
                        const tokenValid = acc.token_valid ?? acc.runtime_probe?.token_valid ?? acc.token_status?.valid
                        const visionCapability = capabilities.vision
                        const cooldownRemaining = numberValue(health.cooldown_remaining)
                        const failureCount = numberValue(health.failure_count)
                        const lastFailureKind = String(health.last_failure_kind || '').trim()
                        const muted = Boolean(acc.muted || health.muted)
                        return (
                            <div key={i} className="p-4 flex flex-col md:flex-row md:items-center justify-between gap-4 hover:bg-muted/50 transition-colors">
                                <div className="flex items-center gap-3 min-w-0">
                                    <div className={clsx(
                                        "w-2 h-2 rounded-full shrink-0",
                                        muted ? "bg-slate-400" :
                                        acc.test_status === 'failed' ? "bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.5)]" :
                                        isActive ? "bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]" :
                                        runtimeUnknown ? "bg-blue-500 shadow-[0_0_8px_rgba(59,130,246,0.5)]" : "bg-amber-500"
                                    )} />
                                    <div className="min-w-0">
                                        <div className="text-sm font-medium truncate">{acc.name || '-'}</div>
                                        <div
                                            className="font-medium truncate flex items-center gap-1.5 cursor-pointer hover:text-primary transition-colors group"
                                            onClick={() => copyId(id)}
                                        >
                                            <span className="truncate">{id || '-'}</span>
                                            {copiedId === id
                                                ? <Check className="w-3 h-3 text-emerald-500 shrink-0" />
                                                : <Copy className="w-3 h-3 opacity-0 group-hover:opacity-50 shrink-0 transition-opacity" />
                                            }
                                        </div>
                                        {acc.remark && (
                                            <div className="text-xs text-muted-foreground truncate mt-0.5">{acc.remark}</div>
                                        )}
                                        <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground mt-0.5 max-w-full">
                                            <span>{acc.test_status === 'failed' ? t('accountManager.testStatusFailed') : isActive ? t('accountManager.sessionActive') : runtimeUnknown ? t('accountManager.runtimeStatusUnknown') : t('accountManager.reauthRequired')}</span>
                                            {acc.token_preview && (
                                                <span className="font-mono bg-muted px-1.5 py-0.5 rounded text-[10px]">
                                                    {acc.token_preview}
                                                </span>
                                            )}
                                            {sessionCounts && sessionCounts[id] !== undefined && (
                                                <span className="font-mono bg-blue-500/10 text-blue-500 px-1.5 py-0.5 rounded text-[10px]">
                                                    {t('accountManager.sessionCount', { count: sessionCounts[id] })}
                                                </span>
                                            )}
                                            {sessionCounts && sessionCounts[id] !== undefined && sessionCounts[id] > 0 && (
                                                <button
                                                    onClick={() => onDeleteAllSessions(id)}
                                                    disabled={deletingSessions && deletingSessions[id]}
                                                    className="flex items-center gap-1 font-mono bg-red-500/10 text-red-500 hover:bg-red-500/20 px-1.5 py-0.5 rounded text-[10px] transition-colors disabled:opacity-50"
                                                    title={t('accountManager.deleteAllSessions')}
                                                >
                                                    {deletingSessions && deletingSessions[id] ? (
                                                        <span className="animate-spin">⟳</span>
                                                    ) : (
                                                        <FolderX className="w-3 h-3" />
                                                    )}
                                                </button>
                                            )}
                                            {acc.proxy_id && (
                                                <span className="font-mono bg-amber-500/10 text-amber-500 px-1.5 py-0.5 rounded text-[10px]">
                                                    {t('accountManager.proxyBadge', { name: assignedProxy ? (assignedProxy.name || `${assignedProxy.host}:${assignedProxy.port}`) : acc.proxy_id })}
                                                </span>
                                            )}
                                            {(tokenValid === true || tokenValid === false || acc.token_checked_at) && (
                                                <span className={clsx(
                                                    "inline-flex items-center gap-1 font-mono px-1.5 py-0.5 rounded text-[10px]",
                                                    tokenValid === false ? "bg-red-500/10 text-red-500" :
                                                    tokenValid === true ? "bg-emerald-500/10 text-emerald-500" :
                                                    "bg-muted text-muted-foreground"
                                                )}>
                                                    <ShieldCheck className="w-3 h-3" />
                                                    {tokenValid === false ? t('accountManager.tokenInvalid') : tokenValid === true ? t('accountManager.tokenValid') : t('accountManager.tokenUnknown')}
                                                </span>
                                            )}
                                            {(visionCapability === true || visionCapability === false || capabilities.checked_at) && (
                                                <span className={clsx(
                                                    "inline-flex items-center gap-1 font-mono px-1.5 py-0.5 rounded text-[10px]",
                                                    visionCapability === true ? "bg-sky-500/10 text-sky-500" :
                                                    visionCapability === false ? "bg-muted text-muted-foreground" :
                                                    "bg-muted text-muted-foreground"
                                                )}>
                                                    <Eye className="w-3 h-3" />
                                                    {visionCapability === false ? t('accountManager.visionUnavailable') : visionCapability === true ? t('accountManager.visionAvailable') : t('accountManager.visionUnknown')}
                                                </span>
                                            )}
                                            {cooldownRemaining > 0 && (
                                                <span className="inline-flex items-center gap-1 font-mono bg-orange-500/10 text-orange-500 px-1.5 py-0.5 rounded text-[10px]">
                                                    <Timer className="w-3 h-3" />
                                                    {t('accountManager.cooldownSeconds', { seconds: cooldownRemaining })}
                                                </span>
                                            )}
                                            {failureCount > 0 && (
                                                <span className="inline-flex items-center gap-1 font-mono bg-red-500/10 text-red-500 px-1.5 py-0.5 rounded text-[10px]">
                                                    <AlertTriangle className="w-3 h-3" />
                                                    {t('accountManager.failureCount', { count: failureCount })}
                                                </span>
                                            )}
                                            {lastFailureKind && (
                                                <span className="font-mono bg-muted px-1.5 py-0.5 rounded text-[10px]">
                                                    {t('accountManager.lastFailureKind', { kind: lastFailureKind })}
                                                </span>
                                            )}
                                            {muted && (
                                                <span className="font-mono bg-muted px-1.5 py-0.5 rounded text-[10px]">
                                                    {t('accountManager.muted')}
                                                </span>
                                            )}
                                        </div>
                                    </div>
                                </div>
                                <div className="flex items-center gap-2 self-start lg:self-auto ml-5 lg:ml-0">
                                    <select
                                        value={acc.proxy_id || ''}
                                        onChange={e => onUpdateAccountProxy(id, e.target.value)}
                                        disabled={updatingProxy?.[id]}
                                        className="max-w-[180px] px-2.5 py-1.5 text-[10px] lg:text-xs bg-secondary border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
                                    >
                                        <option value="">{t('accountManager.proxyNone')}</option>
                                        {proxies.map(proxy => (
                                            <option key={proxy.id} value={proxy.id}>
                                                {proxy.name || `${proxy.host}:${proxy.port}`}
                                            </option>
                                        ))}
                                    </select>
                                    <button
                                        onClick={() => onEditAccount(acc)}
                                        disabled={!id}
                                        className="p-1 lg:p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-md transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                                        title={id ? t('accountManager.editAccountTitle') : t('accountManager.invalidIdentifier')}
                                    >
                                        <Pencil className="w-3.5 h-3.5 lg:w-4 lg:h-4" />
                                    </button>
                                    <button
                                        onClick={() => onTestAccount(id)}
                                        disabled={testing[id]}
                                        className="inline-flex items-center gap-1 px-2 lg:px-3 py-1 lg:py-1.5 text-[10px] lg:text-xs font-medium border border-border rounded-md hover:bg-secondary transition-colors disabled:opacity-50"
                                    >
                                        {testing[id] ? <Loader2 className="w-3 h-3 animate-spin" /> : <Play className="w-3 h-3" />}
                                        <span>{testing[id] ? t('actions.testing') : t('actions.test')}</span>
                                    </button>
                                    <button
                                        onClick={() => onDeleteAccount(id)}
                                        className="p-1 lg:p-1.5 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-md transition-colors"
                                    >
                                        <Trash2 className="w-3.5 h-3.5 lg:w-4 lg:h-4" />
                                    </button>
                                </div>
                            </div>
                        )
                    })
                ) : (
                    <div className="p-8 text-center text-muted-foreground">{searchQuery ? t('accountManager.searchNoResults') : t('accountManager.noAccounts')}</div>
                )}
            </div>

            {totalPages > 1 && (
                <div className="p-4 border-t border-border flex items-center justify-between">
                    <div className="flex items-center gap-3">
                        <div className="text-sm text-muted-foreground">
                            {t('accountManager.pageInfo', { current: page, total: totalPages, count: totalAccounts })}
                        </div>
                        <select
                            value={pageSize}
                            onChange={e => onPageSizeChange(Number(e.target.value))}
                            className="text-sm border border-border rounded-md px-2 py-1 bg-background text-foreground"
                        >
                            {[10, 20, 50, 100, 500, 1000, 2000, 5000].map(s => (
                                <option key={s} value={s}>{s}</option>
                            ))}
                        </select>
                    </div>
                    <div className="flex items-center gap-2">
                        <button
                            onClick={onPrevPage}
                            disabled={page <= 1 || loadingAccounts}
                            className="p-2 border border-border rounded-md hover:bg-secondary transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            <ChevronLeft className="w-4 h-4" />
                        </button>
                        <span className="text-sm font-medium px-2">{page} / {totalPages}</span>
                        <button
                            onClick={onNextPage}
                            disabled={page >= totalPages || loadingAccounts}
                            className="p-2 border border-border rounded-md hover:bg-secondary transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            <ChevronRight className="w-4 h-4" />
                        </button>
                    </div>
                </div>
            )}
        </div>
    )
}
