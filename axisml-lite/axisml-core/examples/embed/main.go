// Command embed is a runnable example of hosting axisml-core inside another Go
// program — importing pkg/core, mounting its full HTTP API on the host's own
// server under a path prefix, and driving the module background loops from a
// context the host owns. It is the in-process embedding path described in
// axisml-lite/docs (Part A), the counterpart to running the axisml-core binary
// standalone.
//
// What it demonstrates:
//   - core.New with options (no env, no /etc/axisml/pools, no os.Exit)
//   - an in-memory StaticConfig (the host supplies the default ResourcePool +
//     Tenant instead of reading YAML from disk)
//   - overriding the fixed Lite Settings (bind address is irrelevant here since
//     the host owns the listener; paths / network are the host's to choose)
//   - mounting core.App.Handler under "/axisml" on the host's own mux
//   - starting core.App.Runnables on a cancellable context and shutting down
//     cleanly on SIGINT/SIGTERM
//
// Prerequisites to actually RUN it (the embedding host must provide these — see
// Part C): a reachable Docker daemon (DOCKER_HOST or the default socket) and a
// PostgreSQL the Config below points at. Without them New fails fast. The file
// always compiles, so it doubles as a compile-checked usage reference.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cmv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"

	"github.com/axisml/axisml/axisml-lite/axisml-core/pkg/core"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Cancelled on the first SIGINT/SIGTERM; both the HTTP server and the module
	// background loops are bound to it, so one signal tears the whole thing down.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Config carries the database / OCI / logging settings. The standalone binary
	// reads these from AXISML_ env; an embedding host fills them in directly. Here
	// we point at a host-provided Postgres and registry.
	cfg := core.Config{}
	cfg.Database.Host = envOr("PGHOST", "localhost")
	cfg.Database.Port = 5432
	cfg.Database.Name = envOr("PGDATABASE", "axisml")
	cfg.Database.User = envOr("PGUSER", "axisml")
	cfg.Database.Password = envOr("PGPASSWORD", "axisml")
	cfg.Database.SSLMode = "disable"
	cfg.Log.Level = "info"
	cfg.Log.Format = "console"
	cfg.OCI.Endpoint = envOr("AXISML_OCI_ENDPOINT", "http://localhost:5000")
	cfg.OCI.AdminUser = "admin"
	cfg.OCI.AdminPassword = envOr("AXISML_OCI_ADMIN_PASSWORD", "admin")

	// Settings are the operational parameters that are fixed constants for the
	// Lite binary. Start from the defaults and override only what the host needs.
	// APIBindAddress is left unused because the host owns the listener below.
	settings := core.DefaultSettings()
	settings.StateDir = envOr("AXISML_STATE_DIR", "/tmp/axisml-embed/runtime")
	settings.GatewayConfigDir = envOr("AXISML_GATEWAY_DIR", "/tmp/axisml-embed/traefik")

	// Assemble axisml-core. We supply the static config in memory (so there is no
	// dependency on /etc/axisml/pools) and let New open the database from Config.
	//
	// To reuse a *gorm.DB the host already owns, add core.WithDB(db) — then
	// App.Close leaves it open for the host to close.
	app, err := core.New(ctx, cfg,
		core.WithSettings(settings),
		core.WithStaticConfig(defaultStaticConfig()),
	)
	if err != nil {
		return err
	}
	defer func() { _ = app.Close() }()

	// Bring the schema up to date. The host decides when — here, at startup.
	if err := app.Migrate(); err != nil {
		return err
	}

	// Mount the full axisml-core API under the host's own prefix. core's routes
	// live under /api/v1, so behind StripPrefix they surface at
	// /axisml/api/v1/... alongside whatever else the host serves.
	mux := http.NewServeMux()
	mux.Handle("/axisml/", http.StripPrefix("/axisml", app.Handler()))
	mux.HandleFunc("/host/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("this route belongs to the host, not axisml-core\n"))
	})

	// The host owns the listener and its lifecycle (TLS, timeouts, middleware…).
	srv := &http.Server{Addr: ":9090", Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	// Start the module background loops (Compute reconcilers + status pollers and
	// the Artifact Hub GC) on the host's context. App.Serve would do this for us,
	// but an embedding host that runs its own server starts them by hand instead.
	for _, r := range app.Runnables() {
		go func(run core.Runnable) {
			if err := run.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("axisml runnable exited: %v", err)
			}
		}(r)
	}

	go func() {
		log.Printf("host server listening on %s — axisml-core API under /axisml/api/v1", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("host server error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutCtx)
}

// defaultStaticConfig builds the single default ResourcePool + Tenant in memory,
// mirroring axisml-lite/deploy/config/{resource-pool,tenant}.yaml. New validates
// it the same way it validates the on-disk form, so the Lite invariants
// (everything named "default", quotas reference the default pool, units fit
// within the quota max) still apply.
//
// The simpler alternative is to keep the YAML on disk and point
// Settings.PoolConfigDir at it, dropping WithStaticConfig entirely.
func defaultStaticConfig() *core.StaticConfig {
	pool := &cmv1alpha1.ResourcePool{
		TypeMeta:   metav1.TypeMeta{APIVersion: "axisml.io/v1alpha1", Kind: "ResourcePool"},
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec: cmv1alpha1.ResourcePoolSpec{
			Units: []cmv1alpha1.ResourceUnit{
				{
					Name: "cpu-small",
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
				{
					Name: "cpu-medium",
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
			},
		},
	}

	tenant := &tenantv1alpha1.Tenant{
		TypeMeta:   metav1.TypeMeta{APIVersion: "axisml.io/v1alpha1", Kind: "Tenant"},
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec: tenantv1alpha1.TenantSpec{
			Namespace: tenantv1alpha1.NamespaceSpec{Name: "default"},
			Quotas: []tenantv1alpha1.QuotaSpec{
				{
					Pool: "default",
					Name: "default",
					Min: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0"),
						corev1.ResourceMemory: resource.MustParse("0"),
					},
					Max: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("16"),
						corev1.ResourceMemory: resource.MustParse("64Gi"),
					},
				},
			},
		},
	}

	return &core.StaticConfig{Pool: pool, Tenant: tenant}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
