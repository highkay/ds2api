package client

import (
	"context"
	dsprotocol "ds2api/internal/deepseek/protocol"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"ds2api/internal/account"
	"ds2api/internal/auth"
	"ds2api/internal/config"
)

func (c *Client) Login(ctx context.Context, acc config.Account) (string, error) {
	clients := c.requestClientsForAccount(acc)
	accountID := strings.TrimSpace(acc.Identifier())
	payload := map[string]any{
		"password":  strings.TrimSpace(acc.Password),
		"device_id": DeviceID(accountID),
		"os":        "Android",
	}
	if email := strings.TrimSpace(acc.Email); email != "" {
		payload["email"] = email
	} else if mobile := strings.TrimSpace(acc.Mobile); mobile != "" {
		loginMobile, areaCode := normalizeMobileForLogin(mobile)
		payload["mobile"] = loginMobile
		payload["area_code"] = areaCode
	} else {
		return "", errors.New("missing email/mobile")
	}
	resp, err := c.postJSON(ctx, clients.regular, clients.fallback, dsprotocol.DeepSeekLoginURL, dsprotocol.BaseHeaders, payload)
	if err != nil {
		return "", err
	}
	code := intFrom(resp["code"])
	if code != 0 {
		return "", fmt.Errorf("login failed: %v", resp["msg"])
	}
	data, _ := resp["data"].(map[string]any)
	if intFrom(data["biz_code"]) != 0 {
		return "", fmt.Errorf("login failed: %v", data["biz_msg"])
	}
	bizData, _ := data["biz_data"].(map[string]any)
	user, _ := bizData["user"].(map[string]any)
	token, _ := user["token"].(string)
	if strings.TrimSpace(token) == "" {
		return "", errors.New("missing login token")
	}
	return token, nil
}

func (c *Client) CreateSession(ctx context.Context, a *auth.RequestAuth, maxAttempts int) (string, error) {
	if maxAttempts <= 0 {
		maxAttempts = c.maxRetries
	}
	clients := c.requestClientsForAuth(ctx, a)
	attempts := 0
	refreshed := false
	for attempts < maxAttempts {
		headers := c.authHeaders(a.DeepSeekToken)
		resp, status, err := c.postJSONWithStatus(ctx, clients.regular, clients.fallback, dsprotocol.DeepSeekCreateSessionURL, headers, map[string]any{"agent": "chat"})
		if err != nil {
			config.Logger.Warn("[create_session] request error", "error", err, "account", a.AccountID)
			if switchAccountAfterPenalty(ctx, a, account.PenaltyNetwork) {
				refreshed = false
				attempts++
				clients = c.requestClientsForAuth(ctx, a)
				continue
			}
			return "", err
		}
		code, bizCode, msg, bizMsg := extractResponseStatus(resp)
		if muted, muteErr := c.handleMutedResponse(ctx, a, "create session", resp); muted {
			if muteErr != nil {
				return "", muteErr
			}
			refreshed = false
			attempts++
			clients = c.requestClientsForAuth(ctx, a)
			continue
		}
		if status == http.StatusOK && code == 0 && bizCode == 0 {
			sessionID := extractCreateSessionID(resp)
			if sessionID != "" {
				return sessionID, nil
			}
		}
		config.Logger.Warn("[create_session] failed", "status", status, "code", code, "biz_code", bizCode, "msg", msg, "biz_msg", bizMsg, "use_config_token", a.UseConfigToken, "account", a.AccountID)
		if a.UseConfigToken {
			if !refreshed && shouldAttemptRefresh(status, code, bizCode, msg, bizMsg) {
				if c.Auth.RefreshToken(ctx, a) {
					refreshed = true
					continue
				}
			}
			penalty := penaltyForFailedStatus(status, code, bizCode, msg, bizMsg)
			if switchAccountAfterPenalty(ctx, a, penalty) {
				refreshed = false
				attempts++
				clients = c.requestClientsForAuth(ctx, a)
				continue
			}
			if penalty != account.PenaltyUnknown {
				return "", fmt.Errorf("create session failed: status=%d code=%d biz_code=%d msg=%s biz_msg=%s", status, code, bizCode, msg, bizMsg)
			}
		}
		attempts++
	}
	return "", errors.New("create session failed")
}

func (c *Client) GetPow(ctx context.Context, a *auth.RequestAuth, maxAttempts int) (string, error) {
	return c.GetPowForTarget(ctx, a, dsprotocol.DeepSeekCompletionTargetPath, maxAttempts)
}

func (c *Client) GetPowForTarget(ctx context.Context, a *auth.RequestAuth, targetPath string, maxAttempts int) (string, error) {
	if maxAttempts <= 0 {
		maxAttempts = c.maxRetries
	}
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		targetPath = dsprotocol.DeepSeekCompletionTargetPath
	}
	clients := c.requestClientsForAuth(ctx, a)
	attempts := 0
	refreshed := false
	lastFailureKind := FailureUnknown
	lastFailureMessage := ""
	for attempts < maxAttempts {
		headers := c.authHeaders(a.DeepSeekToken)
		resp, status, err := c.postJSONWithStatus(ctx, clients.regular, clients.fallback, dsprotocol.DeepSeekCreatePowURL, headers, map[string]any{"target_path": targetPath})
		if err != nil {
			config.Logger.Warn("[get_pow] request error", "error", err, "account", a.AccountID, "target_path", targetPath)
			lastFailureKind = FailureUnknown
			lastFailureMessage = err.Error()
			if switchAccountAfterPenalty(ctx, a, account.PenaltyNetwork) {
				refreshed = false
				attempts++
				clients = c.requestClientsForAuth(ctx, a)
				continue
			}
			return "", err
		}
		code, bizCode, msg, bizMsg := extractResponseStatus(resp)
		if muted, muteErr := c.handleMutedResponse(ctx, a, "get pow", resp); muted {
			if muteErr != nil {
				return "", muteErr
			}
			refreshed = false
			attempts++
			clients = c.requestClientsForAuth(ctx, a)
			continue
		}
		if status == http.StatusOK && code == 0 && bizCode == 0 {
			data, _ := resp["data"].(map[string]any)
			bizData, _ := data["biz_data"].(map[string]any)
			challenge, _ := bizData["challenge"].(map[string]any)
			answer, err := ComputePow(ctx, challenge)
			if err != nil {
				attempts++
				continue
			}
			return BuildPowHeader(challenge, answer)
		}
		config.Logger.Warn("[get_pow] failed", "status", status, "code", code, "biz_code", bizCode, "msg", msg, "biz_msg", bizMsg, "use_config_token", a.UseConfigToken, "account", a.AccountID, "target_path", targetPath)
		lastFailureMessage = failureMessage(msg, bizMsg, "get pow failed")
		if isTokenInvalid(status, code, bizCode, msg, bizMsg) || isAuthIndicativeBizFailure(msg, bizMsg) {
			lastFailureKind = authFailureKind(a.UseConfigToken)
		} else {
			lastFailureKind = FailureUnknown
		}
		if a.UseConfigToken {
			if !refreshed && shouldAttemptRefresh(status, code, bizCode, msg, bizMsg) {
				if c.Auth.RefreshToken(ctx, a) {
					refreshed = true
					continue
				}
			}
			penalty := penaltyForFailedStatus(status, code, bizCode, msg, bizMsg)
			if switchAccountAfterPenalty(ctx, a, penalty) {
				refreshed = false
				attempts++
				clients = c.requestClientsForAuth(ctx, a)
				continue
			}
			if penalty != account.PenaltyUnknown {
				return "", &RequestFailure{Op: "get pow", Kind: lastFailureKind, Message: lastFailureMessage}
			}
		}
		attempts++
	}
	if lastFailureKind != FailureUnknown {
		return "", &RequestFailure{Op: "get pow", Kind: lastFailureKind, Message: lastFailureMessage}
	}
	return "", errors.New("get pow failed")
}
