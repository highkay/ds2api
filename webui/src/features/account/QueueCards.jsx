import { CheckCircle2, Server, ShieldCheck } from 'lucide-react'

function formatMetric(value) {
    if (value === undefined || value === null || value === '') return '-'
    return value
}

function MetricRow({ label, value }) {
    return (
        <div className="mt-3 flex items-center justify-between gap-3 text-xs text-muted-foreground">
            <span className="truncate">{label}</span>
            <span className="font-mono text-foreground">{formatMetric(value)}</span>
        </div>
    )
}

export default function QueueCards({ queueStatus, t }) {
    if (!queueStatus) {
        return null
    }

    return (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-card border border-border rounded-xl p-4 flex flex-col justify-between shadow-sm relative overflow-hidden group">
                <div className="absolute right-0 top-0 p-4 opacity-5 group-hover:opacity-10 transition-opacity">
                    <CheckCircle2 className="w-16 h-16" />
                </div>
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-widest">{t('accountManager.available')}</p>
                <div className="mt-2 flex items-baseline gap-2">
                    <span className="text-3xl font-bold text-foreground">{queueStatus.available}</span>
                    <span className="text-xs text-muted-foreground">{t('accountManager.accountsUnit')}</span>
                </div>
                <MetricRow
                    label={t('accountManager.recommendedConcurrency')}
                    value={queueStatus.recommended_concurrency}
                />
            </div>
            <div className="bg-card border border-border rounded-xl p-4 flex flex-col justify-between shadow-sm relative overflow-hidden group">
                <div className="absolute right-0 top-0 p-4 opacity-5 group-hover:opacity-10 transition-opacity">
                    <Server className="w-16 h-16" />
                </div>
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-widest">{t('accountManager.inUse')}</p>
                <div className="mt-2 flex items-baseline gap-2">
                    <span className="text-3xl font-bold text-foreground">{queueStatus.in_use}</span>
                    <span className="text-xs text-muted-foreground">{t('accountManager.threadsUnit')}</span>
                </div>
                <div>
                    <MetricRow
                        label={t('accountManager.waiting')}
                        value={queueStatus.waiting}
                    />
                    <MetricRow
                        label={t('accountManager.queueLimit')}
                        value={queueStatus.max_queue_size}
                    />
                </div>
            </div>
            <div className="bg-card border border-border rounded-xl p-4 flex flex-col justify-between shadow-sm relative overflow-hidden group">
                <div className="absolute right-0 top-0 p-4 opacity-5 group-hover:opacity-10 transition-opacity">
                    <ShieldCheck className="w-16 h-16" />
                </div>
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-widest">{t('accountManager.totalPool')}</p>
                <div className="mt-2 flex items-baseline gap-2">
                    <span className="text-3xl font-bold text-foreground">{queueStatus.total}</span>
                    <span className="text-xs text-muted-foreground">{t('accountManager.accountsUnit')}</span>
                </div>
                <div>
                    <MetricRow
                        label={t('accountManager.healthCheck')}
                        value={queueStatus.health_enabled ? t('accountManager.enabled') : t('accountManager.disabled')}
                    />
                    <MetricRow
                        label={t('accountManager.globalLimit')}
                        value={queueStatus.global_max_inflight}
                    />
                </div>
            </div>
        </div>
    )
}
