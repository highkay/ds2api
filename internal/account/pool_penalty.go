package account

func (p *Pool) PenalizeHealth(accountID string, kind PenaltyKind) {
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
