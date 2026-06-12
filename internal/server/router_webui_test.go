package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdminDeepLinkFallsBackToWebUIIndex(t *testing.T) {
	t.Setenv("DS2API_CONFIG_JSON", `{"keys":["k1"],"accounts":[{"email":"u@example.com","password":"p"}]}`)
	t.Setenv("DS2API_ENV_WRITEBACK", "0")

	staticDir := t.TempDir()
	marker := "admin-spa-index"
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte(marker), 0o644); err != nil {
		t.Fatalf("write admin index: %v", err)
	}
	t.Setenv("DS2API_STATIC_ADMIN_DIR", staticDir)

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/not-a-real-api/deep-link", nil)
	rec := httptest.NewRecorder()
	app.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), marker) {
		t.Fatalf("expected admin index body, got %q", rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store, must-revalidate" {
		t.Fatalf("expected no-store cache header, got %q", got)
	}
}
