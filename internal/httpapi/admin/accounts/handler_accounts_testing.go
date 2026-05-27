package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"ds2api/internal/config"
	"ds2api/internal/prompt"
	"ds2api/internal/promptcompat"
	"ds2api/internal/sse"
)

type modelAliasSnapshotReader struct {
	aliases map[string]string
}

func (m modelAliasSnapshotReader) ModelAliases() map[string]string {
	return m.aliases
}

func (h *Handler) testSingleAccount(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)
	identifier, _ := req["identifier"].(string)
	if strings.TrimSpace(identifier) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "需要账号标识（identifier / email / mobile）"})
		return
	}
	acc, ok := findAccountByIdentifier(h.Store, identifier)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "账号不存在"})
		return
	}
	result := h.testAccount(r.Context(), acc, accountTestOptionsFromRequest(req))
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) testAllAccounts(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)
	opts := accountTestOptionsFromRequest(req)
	accounts := h.Store.Snapshot().Accounts
	if len(accounts) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"total": 0, "success": 0, "failed": 0, "results": []any{}})
		return
	}

	// Concurrent testing with a semaphore to limit parallelism.
	const maxConcurrency = 5
	results := runAccountTestsConcurrently(accounts, maxConcurrency, func(_ int, account config.Account) map[string]any {
		return h.testAccount(r.Context(), account, opts)
	})

	success := 0
	for _, res := range results {
		if ok, _ := res["success"].(bool); ok {
			success++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": len(accounts), "success": success, "failed": len(accounts) - success, "results": results})
}

type accountTestOptions struct {
	Model             string
	Message           string
	Mode              string
	ProbeCapabilities bool
}

func accountTestOptionsFromRequest(req map[string]any) accountTestOptions {
	model, _ := req["model"].(string)
	message, _ := req["message"].(string)
	mode, _ := req["mode"].(string)
	return normalizeAccountTestOptions(accountTestOptions{
		Model:             model,
		Message:           message,
		Mode:              mode,
		ProbeCapabilities: boolFromAny(req["probe_capabilities"]),
	})
}

func normalizeAccountTestOptions(opts accountTestOptions) accountTestOptions {
	opts.Model = strings.TrimSpace(opts.Model)
	if opts.Model == "" {
		opts.Model = "deepseek-v4-flash"
	}
	opts.Message = strings.TrimSpace(opts.Message)
	opts.Mode = strings.ToLower(strings.TrimSpace(opts.Mode))
	switch opts.Mode {
	case "token", "session", "message":
	default:
		if opts.Message != "" {
			opts.Mode = "message"
		} else {
			opts.Mode = "session"
		}
	}
	if opts.Mode == "message" && opts.Message == "" {
		opts.Message = "你好"
	}
	return opts
}

func runAccountTestsConcurrently(accounts []config.Account, maxConcurrency int, testFn func(int, config.Account) map[string]any) []map[string]any {
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}
	sem := make(chan struct{}, maxConcurrency)
	results := make([]map[string]any, len(accounts))
	var wg sync.WaitGroup
	for i, acc := range accounts {
		wg.Add(1)
		go func(idx int, account config.Account) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release
			results[idx] = testFn(idx, account)
		}(i, acc)
	}
	wg.Wait()
	return results
}

func (h *Handler) testAccount(ctx context.Context, acc config.Account, opts accountTestOptions) map[string]any {
	start := time.Now()
	opts = normalizeAccountTestOptions(opts)
	identifier := acc.Identifier()
	runtimeProbe := config.AccountRuntimeProbe{}
	result := map[string]any{
		"account":         identifier,
		"success":         false,
		"response_time":   0,
		"message":         "",
		"model":           opts.Model,
		"mode":            opts.Mode,
		"session_count":   0,
		"config_writable": !h.Store.IsEnvBacked(),
	}
	defer func() {
		status := "failed"
		if success, _ := result["success"].(bool); success {
			status = "ok"
		}
		_ = h.Store.UpdateAccountTestStatus(identifier, status)
		if runtimeProbeHasData(runtimeProbe) {
			_ = h.Store.UpdateAccountRuntimeProbe(identifier, runtimeProbe)
		}
	}()

	if opts.Mode == "token" {
		return h.testAccountTokenMode(ctx, acc, opts, result, &runtimeProbe, start)
	}

	token, err := h.DS.Login(ctx, acc)
	if err != nil {
		result["message"] = "登录失败: " + err.Error()
		return result
	}
	if err := h.Store.UpdateAccountToken(acc.Identifier(), token); err != nil {
		result["message"] = "登录成功但写入运行时 token 失败: " + err.Error()
		return result
	}
	acc.Token = token
	proxyCtx, authCtx := accountProbeContext(ctx, acc, identifier, token)
	if _, tokenProbeErr := h.attachTokenProbe(proxyCtx, token, result, &runtimeProbe); tokenProbeErr != nil {
		result["token_probe_error"] = tokenProbeErr.Error()
	}
	sessionID, err := h.DS.CreateSession(proxyCtx, authCtx, 1)
	if err != nil {
		newToken, loginErr := h.DS.Login(proxyCtx, acc)
		if loginErr != nil {
			result["message"] = "创建会话失败: " + err.Error()
			return result
		}
		token = newToken
		acc.Token = token
		authCtx.DeepSeekToken = token
		authCtx.Account = acc
		if err := h.Store.UpdateAccountToken(acc.Identifier(), token); err != nil {
			result["message"] = "刷新 token 成功但写入运行时 token 失败: " + err.Error()
			return result
		}
		if _, tokenProbeErr := h.attachTokenProbe(proxyCtx, token, result, &runtimeProbe); tokenProbeErr != nil {
			result["token_probe_error"] = tokenProbeErr.Error()
		}
		sessionID, err = h.DS.CreateSession(proxyCtx, authCtx, 1)
		if err != nil {
			result["message"] = "创建会话失败: " + err.Error()
			return result
		}
	}

	// 获取会话数量
	sessionStats, sessionErr := h.DS.GetSessionCountForToken(proxyCtx, token)
	if sessionErr == nil && sessionStats != nil {
		result["session_count"] = sessionStats.FirstPageCount
	}

	if opts.ProbeCapabilities {
		h.attachCapabilityProbe(proxyCtx, identifier, token, result, &runtimeProbe)
	}

	if opts.Mode != "message" {
		result["success"] = true
		result["message"] = "Token 刷新成功（登录与会话创建成功）"
		result["response_time"] = int(time.Since(start).Milliseconds())
		return result
	}
	model := opts.Model
	thinking, search, ok := config.GetModelConfig(model)
	resolvedModel, resolved := config.ResolveModel(modelAliasSnapshotReader{
		aliases: h.Store.Snapshot().ModelAliases,
	}, model)
	if resolved {
		model = resolvedModel
		thinking, search, ok = config.GetModelConfig(model)
	}
	if !ok {
		thinking, search = false, false
	}
	pow, err := h.DS.GetPow(proxyCtx, authCtx, 1)
	if err != nil {
		result["message"] = "获取 PoW 失败: " + err.Error()
		return result
	}
	payload := promptcompat.StandardRequest{
		ResolvedModel: model,
		FinalPrompt:   prompt.MessagesPrepare([]map[string]any{{"role": "user", "content": opts.Message}}),
		Thinking:      thinking,
		Search:        search,
	}.CompletionPayload(sessionID)
	resp, err := h.DS.CallCompletion(proxyCtx, authCtx, payload, pow, 1)
	if err != nil {
		result["message"] = "请求失败: " + err.Error()
		return result
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		result["message"] = fmt.Sprintf("请求失败: HTTP %d", resp.StatusCode)
		return result
	}
	collected := sse.CollectStream(resp, thinking, true)
	result["success"] = true
	result["response_time"] = int(time.Since(start).Milliseconds())
	if collected.Text != "" {
		result["message"] = collected.Text
	} else {
		result["message"] = "（无回复内容）"
	}
	if collected.Thinking != "" {
		result["thinking"] = collected.Thinking
	}
	return result
}
