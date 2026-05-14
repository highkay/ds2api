package client

import (
	"crypto/sha512"
	"encoding/base64"
	"strings"
)

func DeviceID(accountIdentifier string) string {
	trimmed := strings.TrimSpace(accountIdentifier)
	if trimmed == "" {
		trimmed = "ds2api"
	}
	hash := sha512.Sum512([]byte(trimmed))
	return base64.StdEncoding.EncodeToString(hash[:])
}
