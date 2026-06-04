package client

import (
	"context"

	"ds2api/internal/account"
	"ds2api/internal/auth"
)

func switchAccountAfterPenalty(ctx context.Context, a *auth.RequestAuth, kind account.PenaltyKind) bool {
	if a == nil || !a.UseConfigToken || kind == account.PenaltyUnknown {
		return false
	}
	if !a.ShouldRetryAfterPenalty(kind) {
		a.Penalize(kind)
		return false
	}
	return a.SwitchAccountWithPenalty(ctx, kind)
}
