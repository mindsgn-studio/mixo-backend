package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mindsgn-studio/mixo-backend/internal/config"
)

func authMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	return BasicAuth(cfg)(next)
}

func newProtectedTestServer(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("admin dashboard"))
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	return authMiddleware(cfg, mux)
}

func TestBasicAuth_OpenWhenNoPassword(t *testing.T) {
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: ""}
	srv := newProtectedTestServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 without password configured, got %d", w.Code)
	}
}

func TestBasicAuth_RequiresCredentials(t *testing.T) {
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "hunter2"}
	srv := newProtectedTestServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without credentials, got %d", w.Code)
	}
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate challenge header")
	}
}

func TestBasicAuth_AcceptsValidCredentials(t *testing.T) {
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "hunter2"}
	srv := newProtectedTestServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.SetBasicAuth("admin", "hunter2")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with valid credentials, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "admin dashboard") {
		t.Error("expected dashboard body")
	}
}

func TestBasicAuth_RejectsWrongPassword(t *testing.T) {
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "hunter2"}
	srv := newProtectedTestServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.SetBasicAuth("admin", "wrong")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong password, got %d", w.Code)
	}
}

func TestBasicAuth_SubpathsProtected(t *testing.T) {
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "hunter2"}
	srv := newProtectedTestServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/admin/library", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for /admin/library without auth, got %d", w.Code)
	}
}

func TestBasicAuth_NonAdminPathsOpen(t *testing.T) {
	cfg := &config.Config{AdminUsername: "admin", AdminPassword: "hunter2"}
	srv := newProtectedTestServer(cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for non-admin path without auth, got %d", w.Code)
	}
}

func TestIsAdminPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/admin", true},
		{"/admin/library", true},
		{"/admin/analytics/data", true},
		{"/stream", false},
		{"/health", false},
		{"/adminx", false},
	}
	for _, c := range cases {
		if got := isAdminPath(c.path); got != c.want {
			t.Errorf("isAdminPath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
