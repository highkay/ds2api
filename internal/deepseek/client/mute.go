package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"ds2api/internal/auth"
	"ds2api/internal/config"
)

type muteInfo struct {
	Muted bool
	Until float64
}

func extractMuteInfo(resp map[string]any) muteInfo {
	if resp == nil {
		return muteInfo{}
	}
	_, bizCode, msg, bizMsg := extractResponseStatus(resp)
	data, _ := resp["data"].(map[string]any)
	bizData, _ := data["biz_data"].(map[string]any)
	isMuted := intFrom(bizData["is_muted"]) == 1
	combined := strings.ToLower(strings.TrimSpace(msg) + " " + strings.TrimSpace(bizMsg))
	if bizCode == 5 || isMuted || strings.Contains(combined, "muted") {
		return muteInfo{Muted: true, Until: floatFrom(bizData["mute_until"])}
	}
	return muteInfo{}
}

func floatFrom(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f
	default:
		return 0
	}
}

func (c *Client) handleMutedResponse(ctx context.Context, a *auth.RequestAuth, op string, resp map[string]any) (bool, error) {
	info := extractMuteInfo(resp)
	if !info.Muted {
		return false, nil
	}
	if a != nil && a.UseConfigToken {
		a.MarkAccountMuted(info.Until)
		config.Logger.Warn("[account_mute] upstream muted account", "op", op, "account", a.AccountID, "mute_until", info.Until)
		if a.SwitchAccount(ctx) {
			return true, nil
		}
	}
	msg := "account is muted"
	if info.Until > 0 {
		msg = fmt.Sprintf("%s until %.3f", msg, info.Until)
	}
	return true, &RequestFailure{Op: op, Kind: FailureAccountMuted, Message: msg}
}

func (c *Client) detectCompletionMute(ctx context.Context, a *auth.RequestAuth, resp *http.Response) (bool, error) {
	if resp == nil || resp.Body == nil {
		return false, nil
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	looksLikeJSON := strings.Contains(contentType, "json")
	if !looksLikeJSON && (resp.ContentLength <= 0 || resp.ContentLength > 1<<20 || strings.Contains(contentType, "event-stream")) {
		return false, nil
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return false, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	parsed := map[string]any{}
	if len(body) == 0 || json.Unmarshal(body, &parsed) != nil {
		return false, nil
	}
	return c.handleMutedResponse(ctx, a, "completion", parsed)
}
