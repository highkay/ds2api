package account

import (
	"math/rand"
	"sort"
	"sync"
	"time"

	"ds2api/internal/config"
)

type Pool struct {
	store                  *config.Store
	mu                     sync.Mutex
	queue                  []string
	inUse                  map[string]int
	waiters                []chan struct{}
	maxInflightPerAccount  int
	recommendedConcurrency int
	maxQueueSize           int
	globalMaxInflight      int
	healthCfg              healthConfig
	health                 map[string]*accountHealth
	rng                    *rand.Rand
	now                    func() time.Time
}

func NewPool(store *config.Store) *Pool {
	maxPer := 1
	if store != nil {
		maxPer = store.RuntimeAccountMaxInflight()
	}
	p := &Pool{
		store:                 store,
		inUse:                 map[string]int{},
		maxInflightPerAccount: maxPer,
		health:                map[string]*accountHealth{},
		healthCfg:             DefaultHealthConfig().toInternal(),
		rng:                   rand.New(rand.NewSource(time.Now().UnixNano())),
		now:                   time.Now,
	}
	if store != nil {
		p.healthCfg = LoadHealthConfigFromStore(store).toInternal()
	}
	p.Reset()
	return p
}

func (p *Pool) Reset() {
	var accounts []config.Account
	if p.store != nil {
		accounts = p.store.Accounts()
	}
	sort.SliceStable(accounts, func(i, j int) bool {
		iHas := accounts[i].Token != ""
		jHas := accounts[j].Token != ""
		if iHas == jHas {
			return i < j
		}
		return iHas
	})
	ids := make([]string, 0, len(accounts))
	for _, a := range accounts {
		id := a.Identifier()
		if id != "" && a.IsActive() {
			ids = append(ids, id)
		}
	}
	if p.store != nil {
		p.maxInflightPerAccount = p.store.RuntimeAccountMaxInflight()
	} else {
		p.maxInflightPerAccount = maxInflightFromEnv()
	}
	recommended := defaultRecommendedConcurrency(len(ids), p.maxInflightPerAccount)
	queueLimit := maxQueueFromEnv(recommended)
	globalLimit := recommended
	if p.store != nil {
		queueLimit = p.store.RuntimeAccountMaxQueue(recommended)
		globalLimit = p.store.RuntimeGlobalMaxInflight(recommended)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.drainWaitersLocked()
	p.queue = ids
	p.inUse = map[string]int{}
	p.recommendedConcurrency = recommended
	p.maxQueueSize = queueLimit
	p.globalMaxInflight = globalLimit
	if p.store != nil {
		p.healthCfg = LoadHealthConfigFromStore(p.store).toInternal()
	}
	p.pruneHealthLocked(ids)
	config.Logger.Info(
		"[init_account_queue] initialized",
		"total", len(ids),
		"max_inflight_per_account", p.maxInflightPerAccount,
		"global_max_inflight", p.globalMaxInflight,
		"recommended_concurrency", p.recommendedConcurrency,
		"max_queue_size", p.maxQueueSize,
		"health_enabled", p.healthCfg.enabled,
	)
}

func (p *Pool) Release(accountID string) {
	if accountID == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	count := p.inUse[accountID]
	if count <= 0 {
		return
	}
	if count == 1 {
		delete(p.inUse, accountID)
		p.notifyWaiterLocked()
		return
	}
	p.inUse[accountID] = count - 1
	p.notifyWaiterLocked()
}

func (p *Pool) Penalize(accountID string, kind PenaltyKind) {
	if accountID == "" || kind == PenaltyUnknown {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.healthCfg.enabled {
		return
	}
	h := p.healthLocked(accountID)
	h.applyPenalty(p.healthCfg, kind, p.now())
}

func (p *Pool) RecordSuccess(accountID string) {
	if accountID == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.healthCfg.enabled {
		return
	}
	h := p.healthLocked(accountID)
	h.recordSuccess(p.now())
	p.notifyWaiterLocked()
}

func (p *Pool) ApplyHealthConfig(cfg HealthConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthCfg = cfg.toInternal()
	p.notifyWaiterLocked()
}

func (p *Pool) HealthEnabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.healthCfg.enabled
}

func (p *Pool) healthLocked(accountID string) *accountHealth {
	if p.health == nil {
		p.health = map[string]*accountHealth{}
	}
	h, ok := p.health[accountID]
	if !ok {
		h = newAccountHealth()
		p.health[accountID] = h
	}
	return h
}

func (p *Pool) pruneHealthLocked(currentIDs []string) {
	if len(p.health) == 0 {
		return
	}
	keep := make(map[string]struct{}, len(currentIDs))
	for _, id := range currentIDs {
		keep[id] = struct{}{}
	}
	for id := range p.health {
		if _, ok := keep[id]; !ok {
			delete(p.health, id)
		}
	}
}

func (p *Pool) healthSnapshotLocked(now time.Time) []map[string]any {
	if len(p.queue) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(p.queue))
	ids := append([]string(nil), p.queue...)
	sort.Strings(ids)
	for _, id := range ids {
		h := p.health[id]
		entry := map[string]any{
			"id":                 id,
			"in_use":             p.inUse[id],
			"weight":             1.0,
			"failure_count":      0,
			"cooldown_remaining": 0,
			"last_failure_kind":  "",
			"last_success_at":    int64(0),
			"last_failure_at":    int64(0),
			"active":             true,
			"muted":              false,
			"mute_until":         float64(0),
			"last_used":          float64(0),
		}
		if p.store != nil {
			if acc, ok := p.store.FindAccount(id); ok {
				entry["active"] = acc.IsActive()
				entry["muted"] = acc.IsMuted(now)
				entry["mute_until"] = acc.MuteUntil
				entry["last_used"] = acc.LastUsed
			}
		}
		if h != nil {
			entry["weight"] = roundToTwo(h.effectiveWeight(p.healthCfg, now))
			entry["failure_count"] = h.failureCount
			entry["cooldown_remaining"] = int(h.cooldownRemaining(now).Seconds())
			entry["last_failure_kind"] = string(h.lastFailureKind)
			if !h.lastSuccessAt.IsZero() {
				entry["last_success_at"] = h.lastSuccessAt.Unix()
			}
			if !h.lastFailureAt.IsZero() {
				entry["last_failure_at"] = h.lastFailureAt.Unix()
			}
		}
		out = append(out, entry)
	}
	return out
}

func roundToTwo(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func (p *Pool) Status() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	available := make([]string, 0, len(p.queue))
	inUseAccounts := make([]string, 0, len(p.inUse))
	inUseSlots := 0
	for _, id := range p.queue {
		if p.inUse[id] >= p.maxInflightPerAccount {
			continue
		}
		if p.store != nil {
			if _, ok := p.store.FindAvailableAccount(id, now); !ok {
				continue
			}
		}
		if p.healthCfg.enabled {
			if h := p.health[id]; h != nil && h.cooldownRemaining(now) > 0 {
				continue
			}
		}
		available = append(available, id)
	}
	for id, count := range p.inUse {
		if count > 0 {
			inUseAccounts = append(inUseAccounts, id)
			inUseSlots += count
		}
	}
	sort.Strings(inUseAccounts)
	total := len(p.queue)
	if p.store != nil {
		total = len(p.store.Accounts())
	}
	return map[string]any{
		"available":                len(available),
		"in_use":                   inUseSlots,
		"total":                    total,
		"available_accounts":       available,
		"in_use_accounts":          inUseAccounts,
		"max_inflight_per_account": p.maxInflightPerAccount,
		"global_max_inflight":      p.globalMaxInflight,
		"recommended_concurrency":  p.recommendedConcurrency,
		"waiting":                  len(p.waiters),
		"max_queue_size":           p.maxQueueSize,
		"health_enabled":           p.healthCfg.enabled,
		"accounts":                 p.healthSnapshotLocked(now),
	}
}
