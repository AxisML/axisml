package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("RESYNC_PERIOD", "")
	t.Setenv("NAMESPACE_DENYLIST", "")

	cfg := Load()
	if cfg.ResyncPeriod != defaultResyncPeriod {
		t.Errorf("ResyncPeriod = %v; want %v", cfg.ResyncPeriod, defaultResyncPeriod)
	}
	if len(cfg.NamespaceDenylist) == 0 {
		t.Error("expected default denylist to be non-empty")
	}
}

func TestLoad_ResyncPeriodOverride(t *testing.T) {
	t.Setenv("RESYNC_PERIOD", "30s")
	cfg := Load()
	if cfg.ResyncPeriod != 30*time.Second {
		t.Errorf("ResyncPeriod = %v; want 30s", cfg.ResyncPeriod)
	}
}

func TestLoad_BadResyncPeriodFallsBack(t *testing.T) {
	t.Setenv("RESYNC_PERIOD", "garbage")
	cfg := Load()
	if cfg.ResyncPeriod != defaultResyncPeriod {
		t.Errorf("garbage RESYNC_PERIOD should fall back to default; got %v", cfg.ResyncPeriod)
	}
}

func TestLoad_NegativeResyncPeriodFallsBack(t *testing.T) {
	t.Setenv("RESYNC_PERIOD", "-1s")
	cfg := Load()
	if cfg.ResyncPeriod != defaultResyncPeriod {
		t.Errorf("negative duration should fall back to default; got %v", cfg.ResyncPeriod)
	}
}

func TestLoad_DenylistOverride(t *testing.T) {
	t.Setenv("NAMESPACE_DENYLIST", "foo,bar, baz ,")
	cfg := Load()
	if len(cfg.NamespaceDenylist) != 3 {
		t.Errorf("len(denylist) = %d; want 3", len(cfg.NamespaceDenylist))
	}
	for _, n := range []string{"foo", "bar", "baz"} {
		if _, ok := cfg.NamespaceDenylist[n]; !ok {
			t.Errorf("missing %q in denylist: %v", n, cfg.NamespaceDenylist)
		}
	}
}

func TestParseDenylist_TrimsAndDeduplicates(t *testing.T) {
	got := parseDenylist(" a , a , b ")
	if len(got) != 2 {
		t.Errorf("len = %d; want 2 (dedup)", len(got))
	}
}
