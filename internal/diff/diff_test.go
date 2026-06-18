package diff_test

import (
	"testing"

	"github.com/mickamy/culprit/internal/diff"
	"github.com/mickamy/culprit/internal/profile"
)

func frame(fn, file string, line int64) profile.Frame {
	return profile.Frame{Function: fn, File: file, Line: line}
}

func sample(value int64, stack ...profile.Frame) profile.Sample {
	return profile.Sample{Stack: stack, Value: value}
}

func find(t *testing.T, changes []diff.Change, fn string) diff.Change {
	t.Helper()

	for _, c := range changes {
		if c.Function == fn {
			return c
		}
	}

	t.Fatalf("no change for %q in %+v", fn, changes)

	return diff.Change{}
}

func TestRankNamesRegressedFunctionFirst(t *testing.T) {
	t.Parallel()

	mainFrame := frame("main.main", "main.go", 1)

	base := []profile.Sample{
		sample(100, frame("pkg.A", "a.go", 10), mainFrame),
	}
	// head keeps A's own cost the same but adds a new expensive callee B.
	head := []profile.Sample{
		sample(100, frame("pkg.A", "a.go", 10), mainFrame),
		sample(500, frame("pkg.B", "b.go", 20), frame("pkg.A", "a.go", 12), mainFrame),
	}

	changes := diff.Rank(base, head)

	if changes[0].Function != "pkg.B" {
		t.Fatalf("top mover = %q, want pkg.B\n%+v", changes[0].Function, changes)
	}
	if got := changes[0].FlatDelta(); got != 500 {
		t.Errorf("B FlatDelta = %d, want 500", got)
	}
	if changes[0].Line != 20 {
		t.Errorf("B line = %d, want 20", changes[0].Line)
	}

	a := find(t, changes, "pkg.A")
	if got := a.FlatDelta(); got != 0 {
		t.Errorf("A FlatDelta = %d, want 0 (its own cost is unchanged)", got)
	}
	if got := a.CumDelta(); got != 500 {
		t.Errorf("A CumDelta = %d, want 500 (the regression flows through it)", got)
	}
}

func TestRankReportsImprovementsAsNegativeDelta(t *testing.T) {
	t.Parallel()

	mainFrame := frame("main.main", "main.go", 1)

	base := []profile.Sample{sample(100, frame("pkg.A", "a.go", 10), mainFrame)}
	head := []profile.Sample{sample(40, frame("pkg.A", "a.go", 10), mainFrame)}

	changes := diff.Rank(base, head)

	if changes[0].Function != "pkg.A" {
		t.Fatalf("top mover = %q, want pkg.A", changes[0].Function)
	}
	if got := changes[0].FlatDelta(); got != -60 {
		t.Errorf("A FlatDelta = %d, want -60", got)
	}
}

func TestRankDoesNotInflateCumOnRecursion(t *testing.T) {
	t.Parallel()

	head := []profile.Sample{
		sample(100, frame("pkg.R", "r.go", 5), frame("pkg.R", "r.go", 5), frame("main.main", "m.go", 1)),
	}

	changes := diff.Rank(nil, head)

	r := find(t, changes, "pkg.R")
	if r.HeadCum != 100 {
		t.Errorf("R HeadCum = %d, want 100 (recursion counted once)", r.HeadCum)
	}
	if r.HeadFlat != 100 {
		t.Errorf("R HeadFlat = %d, want 100", r.HeadFlat)
	}
}

func TestRankLocalizesToDominantLine(t *testing.T) {
	t.Parallel()

	mainFrame := frame("main.main", "main.go", 1)

	// C runs as a leaf at line 7 (weight 30) but mostly appears as the caller of
	// D at line 9 (weight 70), so its dominant line is 9.
	head := []profile.Sample{
		sample(30, frame("pkg.C", "c.go", 7), mainFrame),
		sample(70, frame("pkg.D", "d.go", 1), frame("pkg.C", "c.go", 9), mainFrame),
	}

	changes := diff.Rank(nil, head)

	if changes[0].Function != "pkg.D" {
		t.Fatalf("top mover = %q, want pkg.D", changes[0].Function)
	}

	c := find(t, changes, "pkg.C")
	if c.Line != 9 {
		t.Errorf("C line = %d, want 9 (dominant cum line)", c.Line)
	}
}
