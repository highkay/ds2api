package client

import (
	"bytes"
	"context"
	dsprotocol "ds2api/internal/deepseek/protocol"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"ds2api/internal/account"
	"ds2api/internal/auth"
	"ds2api/internal/config"
	trans "ds2api/internal/deepseek/transport"
)

func (c *Client) CallCompletion(ctx context.Context, a *auth.RequestAuth, payload map[string]any, powResp string, maxAttempts int) (*http.Response, error) {
	if maxAttempts <= 0 {
		maxAttempts = c.maxRetries
	}
	clients := c.requestClientsForAuth(ctx, a)
	headers := c.authHeaders(a.DeepSeekToken)
	headers["x-ds-pow-response"] = powResp
	captureSession := c.capture.Start("deepseek_completion", dsprotocol.DeepSeekCompletionURL, a.AccountID, payload)
	attempts := 0
	for attempts < maxAttempts {
		c.attachHifLeimHeader(ctx, a, headers)
		resp, err := c.streamPost(ctx, clients.stream, dsprotocol.DeepSeekCompletionURL, headers, payload)
		if err != nil {
			if switchAccountAfterPenalty(ctx, a, account.PenaltyNetwork) {
				nextPow, powErr := c.GetPow(ctx, a, maxAttempts)
				if powErr != nil {
					return nil, powErr
				}
				clients = c.requestClientsForAuth(ctx, a)
				headers = c.authHeaders(a.DeepSeekToken)
				headers["x-ds-pow-response"] = nextPow
				powResp = nextPow
				attempts++
				time.Sleep(time.Second)
				continue
			}
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			muted, muteErr := c.detectCompletionMute(ctx, a, resp)
			if muteErr != nil {
				_ = resp.Body.Close()
				return nil, muteErr
			}
			if muted {
				_ = resp.Body.Close()
				nextPow, powErr := c.GetPow(ctx, a, maxAttempts)
				if powErr != nil {
					return nil, powErr
				}
				attempts++
				clients = c.requestClientsForAuth(ctx, a)
				headers = c.authHeaders(a.DeepSeekToken)
				headers["x-ds-pow-response"] = nextPow
				powResp = nextPow
				continue
			}
			if captureSession != nil {
				resp.Body = captureSession.WrapBody(resp.Body, resp.StatusCode)
			}
			resp = c.wrapCompletionWithAutoContinue(ctx, a, payload, powResp, resp)
			return resp, nil
		}
		if captureSession != nil {
			resp.Body = captureSession.WrapBody(resp.Body, resp.StatusCode)
		}
		penalty := completionPenaltyForStatus(resp.StatusCode)
		if penalty == account.PenaltyUnknown {
			return resp, nil
		}
		if switchAccountAfterPenalty(ctx, a, penalty) {
			_ = resp.Body.Close()
			nextPow, powErr := c.GetPow(ctx, a, maxAttempts)
			if powErr != nil {
				return nil, powErr
			}
			clients = c.requestClientsForAuth(ctx, a)
			headers = c.authHeaders(a.DeepSeekToken)
			headers["x-ds-pow-response"] = nextPow
			powResp = nextPow
			attempts++
			time.Sleep(time.Second)
			continue
		}
		return resp, nil
	}
	return nil, errors.New("completion failed")
}

func completionPenaltyForStatus(status int) account.PenaltyKind {
	switch {
	case status == http.StatusTooManyRequests:
		return account.PenaltyHTTP429
	case status == http.StatusForbidden:
		return account.PenaltyHTTP403
	case status >= 500 && status <= 599:
		return account.PenaltyHTTP5xx
	default:
		return account.PenaltyUnknown
	}
}

func (c *Client) streamPost(ctx context.Context, doer trans.Doer, url string, headers map[string]string, payload any) (*http.Response, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	headers = c.jsonHeaders(headers)
	clients := c.requestClientsFromContext(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := doer.Do(req)
	if err != nil {
		config.Logger.Warn("[deepseek] fingerprint stream request failed, fallback to std transport", "url", url, "error", err)
		req2, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
		if reqErr != nil {
			return nil, reqErr
		}
		for k, v := range headers {
			req2.Header.Set(k, v)
		}
		return clients.fallbackS.Do(req2)
	}
	return resp, nil
}
