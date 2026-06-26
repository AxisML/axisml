package axismlconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axisml/axisml/pkg/axismlconfig"
)

type testCfg struct {
	axismlconfig.Common `mapstructure:",squash"`
	Extra               extra `mapstructure:"extra"`
}

type extra struct {
	Endpoint string `mapstructure:"endpoint" default:"http://localhost:5000" doc:"thing endpoint"`
	Token    string `mapstructure:"token" secret:"true" doc:"thing token"`
}

func loadEnvOnly(t *testing.T) testCfg {
	t.Helper()
	var c testCfg
	if err := axismlconfig.Load(&c, axismlconfig.Options{EnvOnly: true}); err != nil {
		t.Fatalf("load: %v", err)
	}
	return c
}

func TestDefaults(t *testing.T) {
	c := loadEnvOnly(t)
	if c.Database.Host != "localhost" || c.Database.Port != 5432 {
		t.Fatalf("db defaults wrong: %+v", c.Database)
	}
	if c.Log.Level != "info" || c.Log.Format != "json" {
		t.Fatalf("log defaults wrong: %+v", c.Log)
	}
	if c.Extra.Endpoint != "http://localhost:5000" {
		t.Fatalf("extra default wrong: %q", c.Extra.Endpoint)
	}
}

func TestEnvOverrideAndWeakTyping(t *testing.T) {
	t.Setenv("AXISML_DATABASE_HOST", "db1")
	t.Setenv("AXISML_DATABASE_PORT", "6000") // string -> int
	c := loadEnvOnly(t)
	if c.Database.Host != "db1" {
		t.Fatalf("host override failed: %q", c.Database.Host)
	}
	if c.Database.Port != 6000 {
		t.Fatalf("port weak-typing failed: %d", c.Database.Port)
	}
}

func TestSecretPlainEnv(t *testing.T) {
	t.Setenv("AXISML_EXTRA_TOKEN", "plain-secret")
	c := loadEnvOnly(t)
	if c.Extra.Token != "plain-secret" {
		t.Fatalf("plain secret env failed: %q", c.Extra.Token)
	}
}

func TestSecretFileOutranksPlainEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	if err := os.WriteFile(path, []byte("  file-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXISML_EXTRA_TOKEN", "plain-secret")
	t.Setenv("AXISML_EXTRA_TOKEN_FILE", path)
	c := loadEnvOnly(t)
	if c.Extra.Token != "file-secret" {
		t.Fatalf("_FILE should outrank plain env and be trimmed, got %q", c.Extra.Token)
	}
}

func TestRedactedMasksSecrets(t *testing.T) {
	t.Setenv("AXISML_EXTRA_TOKEN", "plain-secret")
	c := loadEnvOnly(t)
	dump := axismlconfig.Redacted(&c)
	if want := "extra.token=****"; !contains(dump, want) {
		t.Fatalf("redacted dump should mask secret, got %q", dump)
	}
	if contains(dump, "plain-secret") {
		t.Fatalf("redacted dump leaked secret: %q", dump)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
