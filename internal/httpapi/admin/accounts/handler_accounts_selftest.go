package accounts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	authn "ds2api/internal/auth"
	"ds2api/internal/config"
)

func (h *Handler) testAPI(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)
	model, _ := req["model"].(string)
	message, _ := req["message"].(string)
	apiKey, _ := req["api_key"].(string)
	if model == "" {
		model = "deepseek-v4-flash"
	}
	if message == "" {
		message = "你好"
	}
	if apiKey == "" {
		keys := h.Store.Snapshot().Keys
		if len(keys) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "没有可用的 API Key"})
			return
		}
		apiKey = keys[0]
	}
	host := r.Host
	scheme := "http"
	if strings.Contains(strings.ToLower(host), "vercel") || strings.Contains(strings.ToLower(r.Header.Get("X-Forwarded-Proto")), "https") {
		scheme = "https"
	}
	payload := map[string]any{"model": model, "messages": []map[string]any{{"role": "user", "content": message}}, "stream": false}
	b, _ := json.Marshal(payload)
	request, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, fmt.Sprintf("%s://%s/v1/chat/completions", scheme, host), bytes.NewReader(b))
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(request)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error()})
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			config.Logger.Warn("[admin] close self-test response body failed", "error", err)
		}
	}()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		var parsed any
		_ = json.Unmarshal(body, &parsed)
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "status_code": resp.StatusCode, "response": parsed})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": false, "status_code": resp.StatusCode, "response": string(body)})
}

func (h *Handler) deleteAllSessions(w http.ResponseWriter, r *http.Request) {
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

	authCtx := &authn.RequestAuth{UseConfigToken: false, AccountID: acc.Identifier(), Account: acc}
	proxyCtx := authn.WithAuth(r.Context(), authCtx)
	token, err := h.DS.Login(proxyCtx, acc)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "登录失败: " + err.Error()})
		return
	}
	_ = h.Store.UpdateAccountToken(acc.Identifier(), token)
	authCtx.DeepSeekToken = token

	err = h.DS.DeleteAllSessionsForToken(proxyCtx, token)
	if err != nil {
		newToken, loginErr := h.DS.Login(proxyCtx, acc)
		if loginErr != nil {
			writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "删除失败: " + err.Error()})
			return
		}
		token = newToken
		_ = h.Store.UpdateAccountToken(acc.Identifier(), token)
		authCtx.DeepSeekToken = token
		if retryErr := h.DS.DeleteAllSessionsForToken(proxyCtx, token); retryErr != nil {
			writeJSON(w, http.StatusOK, map[string]any{"success": false, "message": "删除失败: " + retryErr.Error()})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "删除成功"})
}
