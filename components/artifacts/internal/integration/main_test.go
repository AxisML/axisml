//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"

	app "github.com/axisml/axisml/components/artifacts/internal/app"
	"github.com/axisml/axisml/components/artifacts/internal/config"
	"github.com/axisml/axisml/components/artifacts/internal/db"
	"github.com/axisml/axisml/components/artifacts/internal/gc"
	"github.com/axisml/axisml/components/artifacts/internal/server"
)

// suite is the shared per-package test fixture.
type suite struct {
	pgCtr  testcontainers.Container
	gormDB *gorm.DB
	cfg    config.Config
	zot    *fakeZot
	engine *gin.Engine
	gcW    *gc.Worker

	// tenant inserted in setup so handler routes that traverse the resolver
	// have a valid name to use.
	tenantName      string
	tenantNamespace string
}

var (
	once sync.Once
	s    *suite
	skip string
)

// setup spins up the suite once for the package.
func setup(t *testing.T) *suite {
	t.Helper()
	once.Do(func() {
		out, err := bootstrap()
		if err != nil {
			skip = err.Error()
			return
		}
		s = out
	})
	if skip != "" {
		t.Skipf("integration setup unavailable: %s", skip)
	}
	return s
}

func bootstrap() (*suite, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. PostgreSQL via testcontainers.
	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("axisml"),
		tcpostgres.WithUsername("axisml"),
		tcpostgres.WithPassword("axisml"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start pg: %w", err)
	}
	host, err := ctr.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := ctr.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return nil, err
	}

	// 2. fake OCI server stubbing zot's HEAD / DELETE manifest endpoints.
	zot := newFakeZot()

	cfg := config.Config{
		DatabaseHost:     host,
		DatabasePort:     int(port.Num()),
		DatabaseName:     "axisml",
		DatabaseUser:     "axisml",
		DatabasePassword: "axisml",
		DatabaseSSLMode:  "disable",

		APIBindAddress:     ":0",
		ProbesBindAddress:  ":0",
		MetricsBindAddress: ":0",

		LeaderElect: false,

		GCInterval:     time.Second,
		UploadingTTL:   24 * time.Hour,
		UploadTokenTTL: time.Hour,

		OCIEndpoint:      zot.URL.Host,
		OCIScheme:        "http",
		OCIAdminUser:     "admin",
		OCIAdminPassword: "secret",
	}

	// 3. Open DB + run *artifacts* migrations.
	gormDB, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}
	// Compute owns the tenants table; for integration we create a minimal
	// stand-in so the resolver can find a row. The columns mirror what
	// docs/system_design/compute.md §6 calls out.
	if err := gormDB.Exec(`
		CREATE TABLE IF NOT EXISTS tenants (
			id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			name        text NOT NULL,
			namespace   text NOT NULL,
			status      text NOT NULL,
			created_at  timestamptz NOT NULL DEFAULT now(),
			updated_at  timestamptz NOT NULL DEFAULT now(),
			deleted_at  timestamptz
		);
		CREATE EXTENSION IF NOT EXISTS "pgcrypto";
	`).Error; err != nil {
		return nil, fmt.Errorf("seed tenants table: %w", err)
	}
	if err := db.Migrate(gormDB); err != nil {
		return nil, err
	}

	// 4. Insert a test tenant.
	tenantName := "default"
	tenantNamespace := "axisml-default"
	if err := gormDB.Exec(`
		INSERT INTO tenants (name, namespace, status)
		VALUES (?, ?, 'Active')
		ON CONFLICT DO NOTHING
	`, tenantName, tenantNamespace).Error; err != nil {
		return nil, fmt.Errorf("insert tenant: %w", err)
	}

	// 5. BuildModules + assemble the Gin engine.
	log := logr.Discard()
	modules, _, err := app.BuildModules(cfg, gormDB, nil, log)
	if err != nil {
		return nil, fmt.Errorf("build modules: %w", err)
	}
	srv, err := server.New(server.Options{Addr: ":0", Log: log, Modules: modules})
	if err != nil {
		return nil, fmt.Errorf("server.New: %w", err)
	}

	// 6. GC worker with tight tick (ticker not started; tests call Tick).
	w := gc.New(cfg, gormDB, log)

	return &suite{
		pgCtr:           ctr.Container,
		gormDB:          gormDB,
		cfg:             cfg,
		zot:             zot,
		engine:          srv.Engine(),
		gcW:             w,
		tenantName:      tenantName,
		tenantNamespace: tenantNamespace,
	}, nil
}

// TestMain ensures the testcontainer is torn down even on early exit.
func TestMain(m *testing.M) {
	code := m.Run()
	if s != nil && s.pgCtr != nil {
		_ = s.pgCtr.Terminate(context.Background())
	}
	if s != nil && s.zot != nil {
		s.zot.Close()
	}
	os.Exit(code)
}

// fakeZot is an httptest.Server stubbing the bits of OCI v2 the artifacts
// service touches (HEAD + DELETE manifests). It records what cli would
// have pushed via `oras push` so tests can simulate "the cli has uploaded".
type fakeZot struct {
	srv *httptest.Server
	URL *url.URL

	mu        sync.Mutex
	manifests map[string]string // key: <repoPath>:<reference>; value: digest
}

func newFakeZot() *fakeZot {
	z := &fakeZot{manifests: map[string]string{}}
	z.srv = httptest.NewServer(http.HandlerFunc(z.handle))
	u, _ := url.Parse(z.srv.URL)
	z.URL = u
	return z
}

func (z *fakeZot) Close() { z.srv.Close() }

// pushManifest records that a manifest was pushed; the next HEAD will return
// the recorded digest.
func (z *fakeZot) pushManifest(repoPath, reference, digest string) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.manifests[repoPath+":"+reference] = digest
}

func (z *fakeZot) handle(w http.ResponseWriter, r *http.Request) {
	// /v2/<repoPath>/manifests/<reference>
	const prefix = "/v2/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	idx := strings.LastIndex(rest, "/manifests/")
	if idx < 0 {
		http.NotFound(w, r)
		return
	}
	repoPath := rest[:idx]
	reference := rest[idx+len("/manifests/"):]

	z.mu.Lock()
	digest, ok := z.manifests[repoPath+":"+reference]
	z.mu.Unlock()

	switch r.Method {
	case http.MethodHead:
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Docker-Content-Digest", digest)
		w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		// idempotent: NotFound treated as success at the client side too
		z.mu.Lock()
		delete(z.manifests, repoPath+":"+reference)
		z.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	default:
		dump, _ := httputil.DumpRequest(r, false)
		_, _ = fmt.Fprintf(w, "fakeZot: unsupported request:\n%s", dump)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// drive issues a request against the in-process engine.
func (s *suite) drive(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *strings.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = strings.NewReader(string(b))
	}
	var req *http.Request
	if bodyReader != nil {
		req, _ = http.NewRequest(method, path, bodyReader)
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Axisml-User", "e2e-tester")
	rec := httptest.NewRecorder()
	s.engine.ServeHTTP(rec, req)
	return rec
}

func (s *suite) tenantPath(rest string) string {
	return fmt.Sprintf("/api/v1/tenants/%s%s", s.tenantName, rest)
}

// fakeDigest produces a deterministic digest string for testing.
func fakeDigest(seed string) string {
	return "sha256:" + uuidV5(seed)
}

func uuidV5(seed string) string {
	id := uuid.NewSHA1(uuid.NameSpaceURL, []byte(seed))
	return strings.ReplaceAll(id.String(), "-", "") + "0000000000000000000000000000000000"[:64-32]
}
