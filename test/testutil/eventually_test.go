package testutil

import (
	"errors"
	"testing"
	"time"
)

func TestEventually_SucceedsImmediately(t *testing.T) {
	calls := 0
	Eventually(t, time.Second, 10*time.Millisecond, func() error {
		calls++
		return nil
	})
	if calls != 1 {
		t.Fatalf("want 1 call, got %d", calls)
	}
}

func TestEventually_RetriesUntilSuccess(t *testing.T) {
	calls := 0
	Eventually(t, time.Second, 5*time.Millisecond, func() error {
		calls++
		if calls < 3 {
			return errors.New("not yet")
		}
		return nil
	})
	if calls != 3 {
		t.Fatalf("want 3 calls, got %d", calls)
	}
}

func TestUniqueNamespaceName(t *testing.T) {
	cases := []struct {
		test, prefix, wantPrefix string
	}{
		{"TestFoo", "envt-", "envt-testfoo-"},
		{"TestFoo/bar_baz", "e2e-", "e2e-testfoo-bar-baz-"},
		{"TestVeryLongTestNameThatExceedsTheBudgetForSure_AndKeepsGoing", "p-", "p-"},
	}
	for _, c := range cases {
		got := UniqueNamespaceName(c.test, c.prefix)
		if len(got) > 63 {
			t.Fatalf("name too long for input %q: %q (%d)", c.test, got, len(got))
		}
		if got[:len(c.wantPrefix)] != c.wantPrefix {
			t.Fatalf("input %q: want prefix %q, got %q", c.test, c.wantPrefix, got)
		}
	}
}
