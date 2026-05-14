package config

import (
	"strings"
	"time"
)

func (a Account) Identifier() string {
	if strings.TrimSpace(a.Email) != "" {
		return strings.TrimSpace(a.Email)
	}
	if mobile := NormalizeMobileForStorage(a.Mobile); mobile != "" {
		return mobile
	}
	return ""
}

func (a Account) IsActive() bool {
	return a.Active == nil || *a.Active
}

func (a Account) IsMuted(now time.Time) bool {
	if !a.Muted {
		return false
	}
	if a.MuteUntil <= 0 {
		return true
	}
	return a.MuteUntil > float64(now.UnixNano())/1e9
}

func (a Account) MuteExpired(now time.Time) bool {
	return a.Muted && a.MuteUntil > 0 && a.MuteUntil <= float64(now.UnixNano())/1e9
}
