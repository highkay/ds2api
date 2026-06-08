package claude

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ds2api/internal/auth"
)

type countTokensAuthStub struct {
	err error
}

func (s countTokensAuthStub) Determine(_ *http.Request) (*auth.RequestAuth, error) {
	return nil, s.err
}

func (s countTokensAuthStub) DetermineCaller(_ *http.Request) (*auth.RequestAuth, error) {
	return nil, s.err
}

func (countTokensAuthStub) Release(_ *auth.RequestAuth) {}

func TestCountTokensNoAccountReturns429(t *testing.T) {
	h := &Handler{Auth: countTokensAuthStub{err: auth.ErrNoAccount}}
	req := httptest.NewRequest(
		http.MethodPost,
		"/anthropic/v1/messages/count_tokens",
		strings.NewReader(`{"model":"claude-sonnet-4-20250514","messages":[{"role":"user","content":"hi"}]}`),
	)
	rec := httptest.NewRecorder()

	h.CountTokens(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "rate_limit_exceeded" {
		t.Fatalf("unexpected code: %v", errObj["code"])
	}
}
