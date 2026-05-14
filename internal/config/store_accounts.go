package config

import (
	"errors"
	"strings"
	"time"
)

func (s *Store) MarkAccountMuted(identifier string, muteUntil float64) error {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return errors.New("account identifier is required")
	}
	if muteUntil <= 0 {
		muteUntil = float64(time.Now().Add(5 * time.Minute).Unix())
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.findAccountIndexLocked(identifier)
	if !ok {
		return errors.New("account not found")
	}
	s.cfg.Accounts[idx].Muted = true
	s.cfg.Accounts[idx].MuteUntil = muteUntil
	s.cfg.Accounts[idx].LastUsed = float64(time.Now().Unix())
	return s.saveLocked()
}

func (s *Store) ClearAccountMute(identifier string) error {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return errors.New("account identifier is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.findAccountIndexLocked(identifier)
	if !ok {
		return errors.New("account not found")
	}
	s.cfg.Accounts[idx].Muted = false
	s.cfg.Accounts[idx].MuteUntil = 0
	return s.saveLocked()
}

func (s *Store) TouchAccountLastUsed(identifier string, ts float64) error {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return errors.New("account identifier is required")
	}
	if ts <= 0 {
		ts = float64(time.Now().Unix())
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.findAccountIndexLocked(identifier)
	if !ok {
		return errors.New("account not found")
	}
	s.cfg.Accounts[idx].LastUsed = ts
	return s.saveLocked()
}

func (s *Store) FindAvailableAccount(identifier string, now time.Time) (Account, bool) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return Account{}, false
	}
	if now.IsZero() {
		now = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.findAccountIndexLocked(identifier)
	if !ok {
		return Account{}, false
	}
	acc := s.cfg.Accounts[idx]
	if !acc.IsActive() {
		return Account{}, false
	}
	if acc.MuteExpired(now) {
		s.cfg.Accounts[idx].Muted = false
		s.cfg.Accounts[idx].MuteUntil = 0
		acc = s.cfg.Accounts[idx]
		if err := s.saveLocked(); err != nil {
			Logger.Warn("[account_mute] clear expired mute failed", "account", identifier, "error", err)
		}
	}
	if acc.IsMuted(now) {
		return Account{}, false
	}
	return acc, true
}
