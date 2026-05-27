package proxies

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/account"
	"ds2api/internal/auth"
	"ds2api/internal/config"
	dsclient "ds2api/internal/deepseek/client"
	adminconfig "ds2api/internal/httpapi/admin/configmgmt"
	adminshared "ds2api/internal/httpapi/admin/shared"
)

type testingDSMock struct{}

func (m *testingDSMock) Login(_ context.Context, _ config.Account) (string, error) {
	return "token", nil
}
func (m *testingDSMock) CreateSession(_ context.Context, _ *auth.RequestAuth, _ int) (string, error) {
	return "session-id", nil
}
func (m *testingDSMock) GetPow(_ context.Context, _ *auth.RequestAuth, _ int) (string, error) {
	return "pow", nil
}
func (m *testingDSMock) CallCompletion(_ context.Context, _ *auth.RequestAuth, _ map[string]any, _ string, _ int) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}
func (m *testingDSMock) DeleteAllSessionsForToken(_ context.Context, _ string) error { return nil }
func (m *testingDSMock) GetSessionCountForToken(_ context.Context, _ string) (*dsclient.SessionStats, error) {
	return &dsclient.SessionStats{}, nil
}
func (m *testingDSMock) ValidateToken(_ context.Context, token string) (*dsclient.TokenValidationResult, error) {
	return &dsclient.TokenValidationResult{Valid: token != "", HTTPStatus: http.StatusOK}, nil
}
func (m *testingDSMock) GetAccountCapabilities(_ context.Context, _ string, _ string) (*dsclient.AccountCapabilities, error) {
	return &dsclient.AccountCapabilities{Source: "client_settings"}, nil
}

func newHTTPAdminHarness(t *testing.T, rawConfig string, ds adminshared.DeepSeekCaller) http.Handler {
	t.Helper()
	t.Setenv("DS2API_CONFIG_JSON", rawConfig)
	store := config.LoadStore()
	pool := account.NewPool(store)
	h := &Handler{Store: store, Pool: pool, DS: ds}
	configHandler := &adminconfig.Handler{Store: store, Pool: pool, DS: ds}
	r := chi.NewRouter()
	RegisterRoutes(r, h)
	r.Get("/config", configHandler.GetConfig)
	return r
}

func adminReq(method, path string, body []byte) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer admin")
	req.Header.Set("Content-Type", "application/json")
	return req
}
