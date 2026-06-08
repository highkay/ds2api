package responses

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ds2api/internal/auth"
)

type preflightAuthStub struct {
	determineCalls int
	callerCalls    int
	releaseCalls   int
}

func (s *preflightAuthStub) Determine(_ *http.Request) (*auth.RequestAuth, error) {
	s.determineCalls++
	return &auth.RequestAuth{
		UseConfigToken: true,
		DeepSeekToken:  "managed-token",
		CallerID:       "caller:test",
		AccountID:      "acct:test",
		TriedAccounts:  map[string]bool{},
	}, nil
}

func (s *preflightAuthStub) DetermineCaller(_ *http.Request) (*auth.RequestAuth, error) {
	s.callerCalls++
	return &auth.RequestAuth{
		UseConfigToken: true,
		CallerID:       "caller:test",
		TriedAccounts:  map[string]bool{},
	}, nil
}

func (s *preflightAuthStub) Release(_ *auth.RequestAuth) {
	s.releaseCalls++
}

func TestResponsesInvalidJSONDoesNotAcquireAccount(t *testing.T) {
	authStub := &preflightAuthStub{}
	h := &Handler{Auth: authStub}
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	h.Responses(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if authStub.callerCalls != 1 {
		t.Fatalf("expected DetermineCaller once, got %d", authStub.callerCalls)
	}
	if authStub.determineCalls != 0 {
		t.Fatalf("expected no account acquire, got %d Determine calls", authStub.determineCalls)
	}
	if authStub.releaseCalls != 0 {
		t.Fatalf("expected no account release, got %d", authStub.releaseCalls)
	}
}

func TestResponsesOversizedPromptDoesNotAcquireAccount(t *testing.T) {
	authStub := &preflightAuthStub{}
	body, err := json.Marshal(map[string]any{
		"model": "deepseek-v4-flash",
		"input": strings.Repeat("x", 60001),
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	h := &Handler{Auth: authStub}
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()

	h.Responses(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d body=%s", rec.Code, rec.Body.String())
	}
	if authStub.callerCalls != 1 {
		t.Fatalf("expected DetermineCaller once, got %d", authStub.callerCalls)
	}
	if authStub.determineCalls != 0 {
		t.Fatalf("expected no account acquire, got %d Determine calls", authStub.determineCalls)
	}
	if authStub.releaseCalls != 0 {
		t.Fatalf("expected no account release, got %d", authStub.releaseCalls)
	}
}
