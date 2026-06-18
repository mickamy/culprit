package render_test

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/mickamy/culprit/internal/diff"
	"github.com/mickamy/culprit/internal/profile"
	"github.com/mickamy/culprit/internal/render"
)

func change(fn, file string, line, baseFlat, headFlat int64) diff.Change {
	return diff.Change{
		Frame:    profile.Frame{Function: fn, File: file, Line: line},
		BaseFlat: baseFlat,
		HeadFlat: headFlat,
	}
}

func TestDiffSplitsRegressionsAndImprovements(t *testing.T) {
	t.Parallel()

	changes := []diff.Change{
		change("pkg.Slow", "slow.go", 10, 1_000_000, 3_400_000), // +2.4ms
		change("pkg.Fast", "fast.go", 5, 2_000_000, 1_500_000),  // -500µs
	}

	var buf bytes.Buffer
	render.Diff(&buf, changes, render.DiffReport{SampleType: "cpu", Unit: "nanoseconds", Top: 10})
	out := buf.String()

	wants := []string{
		"REGRESSIONS (1)", "pkg.Slow", "slow.go:10", "+2.4ms",
		"IMPROVEMENTS (1)", "pkg.Fast", "-500µs",
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestDiffHumanizesBytes(t *testing.T) {
	t.Parallel()

	changes := []diff.Change{change("pkg.Alloc", "alloc.go", 1, 0, 2*1024*1024)} // +2.0MiB

	var buf bytes.Buffer
	render.Diff(&buf, changes, render.DiffReport{SampleType: "inuse_space", Unit: "bytes", Top: 10})

	if out := buf.String(); !strings.Contains(out, "+2.0MiB") {
		t.Errorf("output missing +2.0MiB\n%s", out)
	}
}

func TestHumanizeHandlesMinInt64(t *testing.T) {
	t.Parallel()

	got := render.Humanize(math.MinInt64, "bytes")

	if strings.HasPrefix(got, "--") {
		t.Fatalf("humanize(MinInt64) = %q, sign doubled by negation overflow", got)
	}
	if got != "-8.0EiB" {
		t.Errorf("humanize(MinInt64, bytes) = %q, want -8.0EiB", got)
	}
}

func TestDiffNoChange(t *testing.T) {
	t.Parallel()

	changes := []diff.Change{change("pkg.Same", "same.go", 1, 500, 500)} // delta 0

	var buf bytes.Buffer
	render.Diff(&buf, changes, render.DiffReport{SampleType: "cpu", Unit: "nanoseconds", Top: 10})

	if out := buf.String(); !strings.Contains(out, "no change") {
		t.Errorf("output missing 'no change'\n%s", out)
	}
}
