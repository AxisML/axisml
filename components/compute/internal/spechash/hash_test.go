package spechash

import "testing"

func TestComputeStable(t *testing.T) {
	a := map[string]any{"k": 1, "j": 2}
	b := map[string]any{"j": 2, "k": 1}
	ah, err := Compute(a)
	if err != nil {
		t.Fatal(err)
	}
	bh, err := Compute(b)
	if err != nil {
		t.Fatal(err)
	}
	if ah != bh {
		t.Errorf("hash differs across map orderings: %s vs %s", ah, bh)
	}
}

func TestComputeDifferent(t *testing.T) {
	ah, _ := Compute(map[string]int{"k": 1})
	bh, _ := Compute(map[string]int{"k": 2})
	if ah == bh {
		t.Error("hash should differ for different inputs")
	}
}
