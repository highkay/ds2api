package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ds2api/internal/auth"
	"ds2api/internal/config"
	dsprotocol "ds2api/internal/deepseek/protocol"
	trans "ds2api/internal/deepseek/transport"
)

const (
	hifLeimTimeout        = 800 * time.Millisecond
	hifLeimFailureBackoff = 30 * time.Second
)

var errHifLeimBackoff = errors.New("hif-leim lookup is in temporary failure backoff")

func (c *Client) attachHifLeimHeader(ctx context.Context, a *auth.RequestAuth, headers map[string]string) {
	if headers == nil {
		return
	}
	delete(headers, "x-hif-leim")
	value, err := c.GetHifLeim(ctx, a)
	if err != nil {
		if !errors.Is(err, errHifLeimBackoff) {
			accountID := ""
			if a != nil {
				accountID = a.AccountID
			}
			config.Logger.Warn("[deepseek] failed to get hif-leim token", "error", err, "account", accountID)
		}
		return
	}
	headers["x-hif-leim"] = value
}

// GetHifLeim fetches the short-lived x-hif-leim header value used by the
// DeepSeek Web completion path. Failures are non-fatal for callers.
func (c *Client) GetHifLeim(ctx context.Context, a *auth.RequestAuth) (string, error) {
	if c == nil {
		return "", errors.New("nil client")
	}
	if c.hifLeimBackoffActive(time.Now()) {
		return "", errHifLeimBackoff
	}
	clients := c.requestClientsForAuth(ctx, a)
	primary, fallback, ok := hifLeimDoers(clients)
	if !ok {
		return "", errors.New("no hif-leim http client available")
	}

	reqCtx, cancel := context.WithTimeout(ctx, hifLeimTimeout)
	defer cancel()

	value, err := c.fetchHifLeim(reqCtx, primary, fallback)
	if err != nil {
		c.markHifLeimFailure(time.Now())
		return "", err
	}
	return value, nil
}

func hifLeimDoers(clients requestClients) (trans.Doer, trans.Doer, bool) {
	if clients.regular == nil && clients.fallback == nil {
		return nil, nil, false
	}
	primary := clients.regular
	if primary == nil {
		primary = clients.fallback
	}
	var fallback trans.Doer
	if clients.fallback != nil {
		fallback = clients.fallback
	} else {
		fallback = primary
	}
	return primary, fallback, true
}

func (c *Client) fetchHifLeim(ctx context.Context, primary trans.Doer, fallback trans.Doer) (string, error) {
	headers := map[string]string{
		"Accept":     "application/json",
		"User-Agent": "DeepSeek/1.8.0 Android/35",
	}
	resp, status, err := c.getJSONWithFallback(ctx, primary, fallback, dsprotocol.DeepSeekHifLeimURL, headers)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("hif-leim query returned status %d", status)
	}
	code, bizCode, msg, bizMsg := extractResponseStatus(resp)
	if code != 0 || bizCode != 0 {
		return "", fmt.Errorf("hif-leim query failed: code=%d biz_code=%d msg=%q biz_msg=%q", code, bizCode, msg, bizMsg)
	}
	data, _ := resp["data"].(map[string]any)
	bizData, _ := data["biz_data"].(map[string]any)
	value, _ := bizData["value"].(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("hif-leim response missing value")
	}
	return value, nil
}

func (c *Client) getJSONWithFallback(ctx context.Context, primary trans.Doer, fallback trans.Doer, url string, headers map[string]string) (map[string]any, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := primary.Do(req)
	if err != nil {
		config.Logger.Warn("[deepseek] hif-leim request failed, fallback to std transport", "url", url, "error", err)
		req2, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if reqErr != nil {
			return nil, 0, reqErr
		}
		for k, v := range headers {
			req2.Header.Set(k, v)
		}
		resp, err = fallback.Do(req2)
		if err != nil {
			return nil, 0, err
		}
	}
	defer func() { _ = resp.Body.Close() }()
	payloadBytes, err := readResponseBody(resp)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	out := map[string]any{}
	if len(payloadBytes) > 0 {
		if err := json.Unmarshal(payloadBytes, &out); err != nil {
			config.Logger.Warn("[deepseek] hif-leim json parse failed", "url", url, "status", resp.StatusCode, "content_encoding", resp.Header.Get("Content-Encoding"), "preview", preview(payloadBytes))
		}
	}
	return out, resp.StatusCode, nil
}

func (c *Client) hifLeimBackoffActive(now time.Time) bool {
	c.hifLeimMu.Lock()
	defer c.hifLeimMu.Unlock()
	return now.Before(c.hifLeimSkipUntil)
}

func (c *Client) markHifLeimFailure(now time.Time) {
	c.hifLeimMu.Lock()
	c.hifLeimSkipUntil = now.Add(hifLeimFailureBackoff)
	c.hifLeimMu.Unlock()
}
