package accounts

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	authn "ds2api/internal/auth"
	"ds2api/internal/config"
	dsclient "ds2api/internal/deepseek/client"
)

func (h *Handler) testAccountTokenMode(ctx context.Context, acc config.Account, opts accountTestOptions, result map[string]any, runtimeProbe *config.AccountRuntimeProbe, start time.Time) map[string]any {
	identifier := acc.Identifier()
	token := strings.TrimSpace(acc.Token)
	if token != "" {
		proxyCtx, _ := accountProbeContext(ctx, acc, identifier, token)
		tokenResult, err := h.attachTokenProbe(proxyCtx, token, result, runtimeProbe)
		if err != nil {
			result["message"] = "Token 验证失败: " + err.Error()
			result["response_time"] = int(time.Since(start).Milliseconds())
			return result
		}
		if tokenResult.Valid {
			if opts.ProbeCapabilities {
				h.attachCapabilityProbe(proxyCtx, identifier, token, result, runtimeProbe)
			}
			result["success"] = true
			result["message"] = "Token 验证成功"
			result["response_time"] = int(time.Since(start).Milliseconds())
			return result
		}
		if strings.TrimSpace(acc.Password) == "" {
			result["message"] = "Token 无效: " + tokenFailureMessage(tokenResult)
			result["response_time"] = int(time.Since(start).Milliseconds())
			return result
		}
	}

	if strings.TrimSpace(acc.Password) == "" {
		if token == "" {
			applyTokenProbeResult(result, runtimeProbe, &dsclient.TokenValidationResult{Valid: false, Message: "empty token"})
		}
		result["message"] = "没有可验证的 token，且账号缺少密码，无法刷新"
		result["response_time"] = int(time.Since(start).Milliseconds())
		return result
	}

	newToken, err := h.DS.Login(ctx, acc)
	if err != nil {
		result["message"] = "登录失败: " + err.Error()
		result["response_time"] = int(time.Since(start).Milliseconds())
		return result
	}
	if err := h.Store.UpdateAccountToken(identifier, newToken); err != nil {
		result["message"] = "登录成功但写入运行时 token 失败: " + err.Error()
		result["response_time"] = int(time.Since(start).Milliseconds())
		return result
	}
	acc.Token = newToken
	proxyCtx, _ := accountProbeContext(ctx, acc, identifier, newToken)
	tokenResult, err := h.attachTokenProbe(proxyCtx, newToken, result, runtimeProbe)
	if err != nil {
		result["message"] = "Token 验证失败: " + err.Error()
		result["response_time"] = int(time.Since(start).Milliseconds())
		return result
	}
	if !tokenResult.Valid {
		result["message"] = "Token 无效: " + tokenFailureMessage(tokenResult)
		result["response_time"] = int(time.Since(start).Milliseconds())
		return result
	}
	if opts.ProbeCapabilities {
		h.attachCapabilityProbe(proxyCtx, identifier, newToken, result, runtimeProbe)
	}
	result["success"] = true
	result["message"] = "Token 刷新并验证成功"
	result["response_time"] = int(time.Since(start).Milliseconds())
	return result
}

func accountProbeContext(ctx context.Context, acc config.Account, identifier, token string) (context.Context, *authn.RequestAuth) {
	authCtx := &authn.RequestAuth{UseConfigToken: false, DeepSeekToken: token, AccountID: identifier, Account: acc}
	return authn.WithAuth(ctx, authCtx), authCtx
}

func (h *Handler) attachTokenProbe(ctx context.Context, token string, result map[string]any, runtimeProbe *config.AccountRuntimeProbe) (*dsclient.TokenValidationResult, error) {
	tokenResult, err := h.DS.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}
	applyTokenProbeResult(result, runtimeProbe, tokenResult)
	return tokenResult, nil
}

func applyTokenProbeResult(result map[string]any, runtimeProbe *config.AccountRuntimeProbe, tokenResult *dsclient.TokenValidationResult) {
	if tokenResult == nil {
		return
	}
	runtimeProbe.TokenValid = boolPtr(tokenResult.Valid)
	runtimeProbe.TokenHTTPStatus = tokenResult.HTTPStatus
	runtimeProbe.TokenCode = tokenResult.Code
	runtimeProbe.TokenBizCode = tokenResult.BizCode
	runtimeProbe.TokenMessage = tokenResult.Message
	runtimeProbe.CheckedAt = time.Now().Unix()
	result["token_valid"] = tokenResult.Valid
	result["token_status"] = tokenStatusResponseMap(*runtimeProbe)
}

func (h *Handler) attachCapabilityProbe(ctx context.Context, identifier, token string, result map[string]any, runtimeProbe *config.AccountRuntimeProbe) {
	capabilities, err := h.DS.GetAccountCapabilities(ctx, token, identifier)
	if err != nil {
		runtimeProbe.CapabilityError = err.Error()
		result["capability_error"] = err.Error()
		return
	}
	capProbe := config.AccountCapabilityProbe{
		Vision:    cloneBoolPtr(capabilities.Vision),
		Models:    slices.Clone(capabilities.Models),
		CheckedAt: capabilities.CheckedAt,
		Source:    capabilities.Source,
	}
	runtimeProbe.Capabilities = capProbe
	result["capabilities"] = capabilityProbeResponseMap(capProbe)
}

func cloneBoolPtr(v *bool) *bool {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func tokenFailureMessage(result *dsclient.TokenValidationResult) string {
	if result == nil {
		return "unknown failure"
	}
	if strings.TrimSpace(result.Message) != "" {
		return result.Message
	}
	if result.HTTPStatus != 0 {
		return fmt.Sprintf("HTTP %d", result.HTTPStatus)
	}
	return "invalid token"
}
