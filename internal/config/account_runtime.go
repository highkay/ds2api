package config

import (
	"errors"
	"slices"
	"strings"
)

type AccountCapabilityProbe struct {
	Vision    *bool    `json:"vision,omitempty"`
	Models    []string `json:"models,omitempty"`
	CheckedAt int64    `json:"checked_at,omitempty"`
	Source    string   `json:"source,omitempty"`
}

func (p AccountCapabilityProbe) Clone() AccountCapabilityProbe {
	out := p
	if p.Vision != nil {
		vision := *p.Vision
		out.Vision = &vision
	}
	out.Models = slices.Clone(p.Models)
	return out
}

type AccountRuntimeProbe struct {
	TokenValid      *bool                  `json:"token_valid,omitempty"`
	TokenHTTPStatus int                    `json:"token_http_status,omitempty"`
	TokenCode       int                    `json:"token_code,omitempty"`
	TokenBizCode    int                    `json:"token_biz_code,omitempty"`
	TokenMessage    string                 `json:"token_message,omitempty"`
	Capabilities    AccountCapabilityProbe `json:"capabilities,omitempty"`
	CapabilityError string                 `json:"capability_error,omitempty"`
	CheckedAt       int64                  `json:"checked_at,omitempty"`
}

func (p AccountRuntimeProbe) Clone() AccountRuntimeProbe {
	out := p
	if p.TokenValid != nil {
		valid := *p.TokenValid
		out.TokenValid = &valid
	}
	out.Capabilities = p.Capabilities.Clone()
	return out
}

func (s *Store) UpdateAccountRuntimeProbe(identifier string, probe AccountRuntimeProbe) error {
	identifier = strings.TrimSpace(identifier)
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.findAccountIndexLocked(identifier)
	if !ok {
		return errors.New("account not found")
	}
	s.setAccountRuntimeProbeLocked(s.cfg.Accounts[idx], probe, identifier)
	return nil
}

func (s *Store) AccountRuntimeProbe(identifier string) (AccountRuntimeProbe, bool) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return AccountRuntimeProbe{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	probe, ok := s.accProbe[identifier]
	if !ok {
		return AccountRuntimeProbe{}, false
	}
	return probe.Clone(), true
}
