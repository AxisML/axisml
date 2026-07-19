package server

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func TestConfigMapNameValidator(t *testing.T) {
	if err := RegisterValidators(); err != nil {
		t.Fatal(err)
	}
	v := binding.Validator.Engine().(*validator.Validate)
	type payload struct {
		ConfigMaps []WorkloadConfigMap `binding:"dive"`
	}
	for _, name := range []string{"config", "trainer.config", "a"} {
		if err := v.Struct(payload{ConfigMaps: []WorkloadConfigMap{{Name: name}}}); err != nil {
			t.Errorf("valid ConfigMap name %q rejected: %v", name, err)
		}
	}
	for _, name := range []string{"Config", "bad_name", ".bad", "bad."} {
		if err := v.Struct(payload{ConfigMaps: []WorkloadConfigMap{{Name: name}}}); err == nil {
			t.Errorf("invalid ConfigMap name %q accepted", name)
		}
	}
}

func newTestServer(t *testing.T, staticFS fs.FS) *Server {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := New(Options{Addr: ":0", Log: log, StaticFS: staticFS})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return srv
}

func do(t *testing.T, srv *Server, pathStr string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, pathStr, nil)
	srv.Engine().ServeHTTP(w, req)
	return w
}

func TestStaticServing(t *testing.T) {
	bundle := fstest.MapFS{
		"index.html":        {Data: []byte("<!doctype html><div id=root></div>")},
		"assets/app.123.js": {Data: []byte("console.log(1)")},
		"favicon.ico":       {Data: []byte("icon")},
	}
	srv := newTestServer(t, bundle)

	// Existing fingerprinted asset → served with an immutable long cache.
	if w := do(t, srv, "/assets/app.123.js"); w.Code != http.StatusOK {
		t.Fatalf("asset: got %d, want 200", w.Code)
	} else if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("asset cache-control = %q", cc)
	}

	// Existing root file → served.
	if w := do(t, srv, "/favicon.ico"); w.Code != http.StatusOK || w.Body.String() != "icon" {
		t.Fatalf("favicon: got %d %q", w.Code, w.Body.String())
	}

	// Unknown non-API route → SPA fallback to index.html (client-side routing).
	w := do(t, srv, "/workspaces/foo")
	if w.Code != http.StatusOK {
		t.Fatalf("spa fallback: got %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("spa fallback content-type = %q", ct)
	}

	// A directory must NOT yield a browsable listing — it falls back to the SPA
	// shell, never exposing bundled filenames.
	for _, p := range []string{"/assets", "/assets/"} {
		w := do(t, srv, p)
		if w.Code == http.StatusOK && strings.Contains(w.Body.String(), "app.123.js") {
			t.Fatalf("dir listing leaked for %s: %q", p, w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Fatalf("dir %s content-type = %q, want SPA html", p, ct)
		}
	}

	// Unknown /api/v1 route → 501, never the SPA shell.
	if w := do(t, srv, "/api/v1/nope"); w.Code != http.StatusNotImplemented {
		t.Fatalf("api 501: got %d, want 501", w.Code)
	}
}

func TestNoStaticFSFallsBackTo404(t *testing.T) {
	srv := newTestServer(t, nil)
	if w := do(t, srv, "/workspaces/foo"); w.Code != http.StatusNotFound {
		t.Fatalf("no static: got %d, want 404", w.Code)
	}
	if w := do(t, srv, "/api/v1/nope"); w.Code != http.StatusNotImplemented {
		t.Fatalf("api 501: got %d, want 501", w.Code)
	}
}
