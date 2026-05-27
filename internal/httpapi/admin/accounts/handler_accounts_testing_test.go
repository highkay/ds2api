package accounts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ds2api/internal/auth"
	"ds2api/internal/config"
	dsclient "ds2api/internal/deepseek/client"
)

type testingDSMock struct {
	loginCalls                 int
	createSessionCalls         int
	getPowCalls                int
	callCompletionCalls        int
	deleteAllSessionsCalls     int
	validateTokenCalls         int
	capabilityCalls            int
	deleteAllSessionsError     error
	deleteAllSessionsErrorOnce bool
}

func (m *testingDSMock) Login(_ context.Context, _ config.Account) (string, error) {
	m.loginCalls++
	return "new-token", nil
}

func (m *testingDSMock) CreateSession(_ context.Context, _ *auth.RequestAuth, _ int) (string, error) {
	m.createSessionCalls++
	return "session-id", nil
}

func (m *testingDSMock) GetPow(_ context.Context, _ *auth.RequestAuth, _ int) (string, error) {
	m.getPowCalls++
	return "", errors.New("should not call GetPow in this test")
}

func (m *testingDSMock) CallCompletion(_ context.Context, _ *auth.RequestAuth, _ map[string]any, _ string, _ int) (*http.Response, error) {
	m.callCompletionCalls++
	return nil, errors.New("should not call CallCompletion in this test")
}

func (m *testingDSMock) DeleteAllSessionsForToken(_ context.Context, _ string) error {
	m.deleteAllSessionsCalls++
	if m.deleteAllSessionsError != nil {
		err := m.deleteAllSessionsError
		if m.deleteAllSessionsErrorOnce {
			m.deleteAllSessionsError = nil
		}
		return err
	}
	return nil
}

func (m *testingDSMock) GetSessionCountForToken(_ context.Context, _ string) (*dsclient.SessionStats, error) {
	return &dsclient.SessionStats{Success: true}, nil
}

func (m *testingDSMock) ValidateToken(_ context.Context, token string) (*dsclient.TokenValidationResult, error) {
	m.validateTokenCalls++
	return &dsclient.TokenValidationResult{Valid: strings.TrimSpace(token) != "", HTTPStatus: http.StatusOK}, nil
}

func (m *testingDSMock) GetAccountCapabilities(_ context.Context, _ string, _ string) (*dsclient.AccountCapabilities, error) {
	m.capabilityCalls++
	vision := true
	return &dsclient.AccountCapabilities{Vision: &vision, Models: []string{"chat", "vision"}, CheckedAt: 123, Source: "client_settings"}, nil
}

func TestTestAccount_BatchModeOnlyCreatesSession(t *testing.T) {
	t.Setenv("DS2API_CONFIG_JSON", `{"accounts":[{"email":"batch@example.com","password":"pwd","token":""}]}`)
	store := config.LoadStore()
	ds := &testingDSMock{}
	h := &Handler{Store: store, DS: ds}
	acc, ok := store.FindAccount("batch@example.com")
	if !ok {
		t.Fatal("expected test account")
	}

	result := h.testAccount(context.Background(), acc, accountTestOptions{Model: "deepseek-v4-flash", Mode: "session"})

	if ok, _ := result["success"].(bool); !ok {
		t.Fatalf("expected success=true, got %#v", result)
	}
	msg, _ := result["message"].(string)
	if !strings.Contains(msg, "Token 刷新成功") {
		t.Fatalf("expected session-only success message, got %q", msg)
	}
	if ds.loginCalls != 1 || ds.createSessionCalls != 1 {
		t.Fatalf("unexpected Login/CreateSession calls: login=%d createSession=%d", ds.loginCalls, ds.createSessionCalls)
	}
	if ds.getPowCalls != 0 || ds.callCompletionCalls != 0 {
		t.Fatalf("expected no completion flow calls, got getPow=%d callCompletion=%d", ds.getPowCalls, ds.callCompletionCalls)
	}
	if ds.validateTokenCalls != 1 || ds.capabilityCalls != 0 {
		t.Fatalf("unexpected probe calls: validate=%d capability=%d", ds.validateTokenCalls, ds.capabilityCalls)
	}
	updated, ok := store.FindAccount("batch@example.com")
	if !ok {
		t.Fatal("expected updated account")
	}
	if updated.Token != "new-token" {
		t.Fatalf("expected refreshed token to be persisted, got %q", updated.Token)
	}
	testStatus, ok := store.AccountTestStatus("batch@example.com")
	if !ok || testStatus != "ok" {
		t.Fatalf("expected runtime test status ok, got %q (ok=%v)", testStatus, ok)
	}
}

func TestTestAccount_ProbeCapabilitiesStoresRuntimeProbe(t *testing.T) {
	t.Setenv("DS2API_CONFIG_JSON", `{"accounts":[{"email":"batch@example.com","password":"pwd","token":""}]}`)
	store := config.LoadStore()
	ds := &testingDSMock{}
	h := &Handler{Store: store, DS: ds}
	acc, ok := store.FindAccount("batch@example.com")
	if !ok {
		t.Fatal("expected test account")
	}

	result := h.testAccount(context.Background(), acc, accountTestOptions{
		Model:             "deepseek-v4-flash",
		Mode:              "session",
		ProbeCapabilities: true,
	})

	if ok, _ := result["success"].(bool); !ok {
		t.Fatalf("expected success=true, got %#v", result)
	}
	if ds.validateTokenCalls != 1 || ds.capabilityCalls != 1 {
		t.Fatalf("expected token and capability probe, got validate=%d capability=%d", ds.validateTokenCalls, ds.capabilityCalls)
	}
	probe, ok := store.AccountRuntimeProbe("batch@example.com")
	if !ok {
		t.Fatal("expected runtime probe cached")
	}
	if probe.TokenValid == nil || !*probe.TokenValid {
		t.Fatalf("expected token valid probe, got %#v", probe)
	}
	if probe.Capabilities.Vision == nil || !*probe.Capabilities.Vision {
		t.Fatalf("expected vision capability, got %#v", probe.Capabilities)
	}
	if !containsString(probe.Capabilities.Models, "vision") {
		t.Fatalf("expected vision model in capability list, got %#v", probe.Capabilities.Models)
	}
}

func TestTestAccount_TokenModeOnlyValidatesExistingToken(t *testing.T) {
	t.Setenv("DS2API_CONFIG_JSON", `{"accounts":[{"email":"batch@example.com","password":"pwd","token":""}]}`)
	store := config.LoadStore()
	if err := store.UpdateAccountToken("batch@example.com", "seed-token"); err != nil {
		t.Fatalf("seed token: %v", err)
	}
	ds := &testingDSMock{}
	h := &Handler{Store: store, DS: ds}
	acc, ok := store.FindAccount("batch@example.com")
	if !ok {
		t.Fatal("expected test account")
	}

	result := h.testAccount(context.Background(), acc, accountTestOptions{
		Model:             "deepseek-v4-flash",
		Mode:              "token",
		ProbeCapabilities: true,
	})

	if ok, _ := result["success"].(bool); !ok {
		t.Fatalf("expected success=true, got %#v", result)
	}
	if ds.loginCalls != 0 || ds.createSessionCalls != 0 || ds.getPowCalls != 0 || ds.callCompletionCalls != 0 {
		t.Fatalf("token mode should not call login/session/completion: login=%d session=%d pow=%d completion=%d", ds.loginCalls, ds.createSessionCalls, ds.getPowCalls, ds.callCompletionCalls)
	}
	if ds.validateTokenCalls != 1 || ds.capabilityCalls != 1 {
		t.Fatalf("expected token and capability probe, got validate=%d capability=%d", ds.validateTokenCalls, ds.capabilityCalls)
	}
	msg, _ := result["message"].(string)
	if !strings.Contains(msg, "Token 验证成功") {
		t.Fatalf("expected token validation success message, got %q", msg)
	}
}

func TestDeleteAllSessions_RetryWithReloginOnDeleteFailure(t *testing.T) {
	t.Setenv("DS2API_CONFIG_JSON", `{"accounts":[{"email":"batch@example.com","password":"pwd","token":"expired-token"}]}`)
	store := config.LoadStore()
	ds := &testingDSMock{deleteAllSessionsError: errors.New("token expired"), deleteAllSessionsErrorOnce: true}
	h := &Handler{Store: store, DS: ds}

	req := httptest.NewRequest(http.MethodPost, "/delete-all", bytes.NewBufferString(`{"identifier":"batch@example.com"}`))
	rec := httptest.NewRecorder()
	h.deleteAllSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if ok, _ := resp["success"].(bool); !ok {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if ds.loginCalls != 2 {
		t.Fatalf("expected initial login plus relogin, got %d", ds.loginCalls)
	}
	if ds.deleteAllSessionsCalls != 2 {
		t.Fatalf("expected delete called twice, got %d", ds.deleteAllSessionsCalls)
	}
	updated, ok := store.FindAccount("batch@example.com")
	if !ok {
		t.Fatal("expected account")
	}
	if updated.Token != "new-token" {
		t.Fatalf("expected refreshed token persisted, got %q", updated.Token)
	}
}

type completionPayloadDSMock struct {
	payload map[string]any
}

func (m *completionPayloadDSMock) Login(_ context.Context, _ config.Account) (string, error) {
	return "new-token", nil
}

func (m *completionPayloadDSMock) CreateSession(_ context.Context, _ *auth.RequestAuth, _ int) (string, error) {
	return "session-id", nil
}

func (m *completionPayloadDSMock) GetPow(_ context.Context, _ *auth.RequestAuth, _ int) (string, error) {
	return "pow-ok", nil
}

func (m *completionPayloadDSMock) CallCompletion(_ context.Context, _ *auth.RequestAuth, payload map[string]any, _ string, _ int) (*http.Response, error) {
	m.payload = payload
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("data: {\"v\":\"ok\"}\n\ndata: [DONE]\n\n")),
	}, nil
}

func (m *completionPayloadDSMock) DeleteAllSessionsForToken(_ context.Context, _ string) error {
	return nil
}

func (m *completionPayloadDSMock) GetSessionCountForToken(_ context.Context, _ string) (*dsclient.SessionStats, error) {
	return &dsclient.SessionStats{Success: true}, nil
}

func (m *completionPayloadDSMock) ValidateToken(_ context.Context, token string) (*dsclient.TokenValidationResult, error) {
	return &dsclient.TokenValidationResult{Valid: strings.TrimSpace(token) != "", HTTPStatus: http.StatusOK}, nil
}

func (m *completionPayloadDSMock) GetAccountCapabilities(_ context.Context, _ string, _ string) (*dsclient.AccountCapabilities, error) {
	return &dsclient.AccountCapabilities{Source: "client_settings"}, nil
}

func TestTestAccount_MessageModeUsesExpertModelTypeForExpertModel(t *testing.T) {
	t.Setenv("DS2API_CONFIG_JSON", `{"accounts":[{"email":"batch@example.com","password":"pwd","token":"seed-token"}]}`)
	store := config.LoadStore()
	ds := &completionPayloadDSMock{}
	h := &Handler{Store: store, DS: ds}
	acc, ok := store.FindAccount("batch@example.com")
	if !ok {
		t.Fatal("expected test account")
	}

	result := h.testAccount(context.Background(), acc, accountTestOptions{Model: "deepseek-v4-pro", Message: "hello", Mode: "message"})

	if ok, _ := result["success"].(bool); !ok {
		t.Fatalf("expected success=true, got %#v", result)
	}
	if got := ds.payload["model_type"]; got != "expert" {
		t.Fatalf("expected model_type expert, got %#v", got)
	}
	if got := ds.payload["chat_session_id"]; got != "session-id" {
		t.Fatalf("unexpected chat_session_id: %#v", got)
	}
}

func TestTestAccount_MessageModeUsesVisionModelTypeForVisionModel(t *testing.T) {
	t.Setenv("DS2API_CONFIG_JSON", `{"accounts":[{"email":"batch@example.com","password":"pwd","token":"seed-token"}]}`)
	store := config.LoadStore()
	ds := &completionPayloadDSMock{}
	h := &Handler{Store: store, DS: ds}
	acc, ok := store.FindAccount("batch@example.com")
	if !ok {
		t.Fatal("expected test account")
	}

	result := h.testAccount(context.Background(), acc, accountTestOptions{Model: "deepseek-v4-vision", Message: "hello", Mode: "message"})

	if ok, _ := result["success"].(bool); !ok {
		t.Fatalf("expected success=true, got %#v", result)
	}
	if got := ds.payload["model_type"]; got != "vision" {
		t.Fatalf("expected model_type vision, got %#v", got)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
