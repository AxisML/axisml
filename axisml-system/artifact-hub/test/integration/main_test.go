//go:build integration

package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/testcontainers/testcontainers-go"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"

	app "github.com/axisml/axisml/axisml-system/artifact-hub/internal/app"
	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/config"
	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/db"
	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/gc"
	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/server"
	"github.com/axisml/axisml/pkg/axismlconfig"
)

// suite is the shared per-package test fixture.
type suite struct {
	pgCtr  testcontainers.Container
	gormDB *gorm.DB
	cfg    config.Config
	zot    *fakeZot
	engine *gin.Engine
	gcW    *gc.Worker

	// s3 backs the dataset kind (real MinIO testcontainer). s3cli seeds the
	// artifact-manifest.json objects the handler verifies against.
	s3Ctr    *tcminio.MinioContainer
	s3cli    *minio.Client
	s3Bucket string

	// namespace is the legacy API name for the opaque logical tenant scope.
	namespace string
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
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
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

	// 3. Real MinIO (S3) via testcontainers, backing the dataset kind so
	// complete verifies the stored artifact-manifest.json digest for real.
	const s3Bucket = "axisml-artifact-hub"
	mc, err := tcminio.Run(ctx, "minio/minio:latest")
	if err != nil {
		return nil, fmt.Errorf("start minio: %w", err)
	}
	s3Endpoint, err := mc.ConnectionString(ctx)
	if err != nil {
		return nil, err
	}

	cfg := config.Config{
		Common: axismlconfig.Common{
			Database: axismlconfig.Database{
				Host:     host,
				Port:     int(port.Num()),
				Name:     "axisml",
				User:     "axisml",
				Password: "axisml",
				SSLMode:  "disable",
			},
			Log: axismlconfig.Log{Level: "info", Format: "console"},
		},
		OCI: config.OCI{
			// host:port with no scheme — the OCI client defaults to http.
			Endpoint:      zot.URL.Host,
			AdminUser:     "admin",
			AdminPassword: "secret",
		},
		S3: config.S3{
			// host:port with no scheme — the S3 client defaults to http.
			Endpoint:  s3Endpoint,
			AccessKey: mc.Username,
			SecretKey: mc.Password,
			Bucket:    s3Bucket,
		},
	}

	// 3. Open DB + run *artifacts* migrations.
	gormDB, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(gormDB); err != nil {
		return nil, err
	}

	// 4. BuildModules + assemble the Gin engine.
	log := logr.Discard()
	modules, _, caps, err := app.BuildModules(cfg, gormDB, log)
	if err != nil {
		return nil, fmt.Errorf("build modules: %w", err)
	}
	srv, err := server.New(server.Options{Addr: ":0", Log: log, Modules: modules, Capabilities: caps})
	if err != nil {
		return nil, fmt.Errorf("server.New: %w", err)
	}

	// GC worker (ticker not started; tests call Tick directly with a fast-
	// forwarded clock). UploadingTTL is the fixed 24h constant.
	w := gc.New(config.GCInterval, config.UploadingTTL, gormDB, log)

	// Seeding client: BuildModules already created the bucket via EnsureBucket,
	// so this client only PUTs the artifact-manifest.json objects the dataset
	// tests verify against.
	s3cli, err := minio.New(s3Endpoint, &minio.Options{
		Creds:  miniocreds.NewStaticV4(mc.Username, mc.Password, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("minio seed client: %w", err)
	}

	return &suite{
		pgCtr:     ctr.Container,
		gormDB:    gormDB,
		cfg:       cfg,
		zot:       zot,
		s3Ctr:     mc,
		s3cli:     s3cli,
		s3Bucket:  s3Bucket,
		engine:    srv.Engine(),
		gcW:       w,
		namespace: "axisml-default",
	}, nil
}

// TestMain ensures the testcontainer is torn down even on early exit.
func TestMain(m *testing.M) {
	code := m.Run()
	if s != nil && s.pgCtr != nil {
		_ = s.pgCtr.Terminate(context.Background())
	}
	if s != nil && s.s3Ctr != nil {
		_ = s.s3Ctr.Terminate(context.Background())
	}
	if s != nil && s.zot != nil {
		s.zot.Close()
	}
	os.Exit(code)
}

// putDatasetManifest uploads an artifact-manifest.json for the dataset version
// and returns its sha256:... digest — the value the client would claim on
// complete. The handler GETs the same object and must hash to this digest.
func (s *suite) putDatasetManifest(t *testing.T, name, version string, body []byte) string {
	t.Helper()
	key := fmt.Sprintf("namespaces/%s/datasets/%s/%s/artifact-manifest.json", s.namespace, name, version)
	_, err := s.s3cli.PutObject(context.Background(), s.s3Bucket, key,
		strings.NewReader(string(body)), int64(len(body)),
		minio.PutObjectOptions{ContentType: "application/json"})
	if err != nil {
		t.Fatalf("put dataset manifest: %v", err)
	}
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
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

// nsPath builds /api/v1/namespaces/{ns}{rest}.
func (s *suite) nsPath(rest string) string {
	return fmt.Sprintf("/api/v1/namespaces/%s%s", s.namespace, rest)
}

// resetState wipes the artifacts table between tests so cases are independent.
func (s *suite) resetState(t *testing.T) {
	t.Helper()
	if err := s.gormDB.Exec("TRUNCATE TABLE artifacts").Error; err != nil {
		t.Fatalf("truncate artifacts: %v", err)
	}
	s.zot.mu.Lock()
	s.zot.manifests = map[string]string{}
	s.zot.mu.Unlock()
}

// fakeDigest produces a deterministic digest string for testing.
func fakeDigest(seed string) string {
	return "sha256:" + uuidV5(seed)
}

func uuidV5(seed string) string {
	id := uuid.NewSHA1(uuid.NameSpaceURL, []byte(seed))
	return strings.ReplaceAll(id.String(), "-", "") + "0000000000000000000000000000000000"[:64-32]
}
