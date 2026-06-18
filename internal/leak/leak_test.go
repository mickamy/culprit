package leak_test

import (
	"math"
	"testing"

	"github.com/mickamy/culprit/internal/leak"
	"github.com/mickamy/culprit/internal/profile"
)

func alloc(site profile.Frame, bytes int64) profile.Sample {
	return profile.Sample{Stack: []profile.Frame{site}, Value: bytes}
}

func snapshot(seconds float64, samples ...profile.Sample) leak.Snapshot {
	return leak.Snapshot{Seconds: seconds, Samples: samples}
}

func find(t *testing.T, growths []leak.Growth, fn string) leak.Growth {
	t.Helper()

	for _, g := range growths {
		if g.Function == fn {
			return g
		}
	}

	t.Fatalf("no growth for %q in %+v", fn, growths)

	return leak.Growth{}
}

func TestFitRecoversSlopeAndPerfectFit(t *testing.T) {
	t.Parallel()

	// ys = 10*x + 10 → slope 10, perfect fit.
	xs := []float64{0, 1, 2, 3}
	ys := []int64{10, 20, 30, 40}

	slope, r2 := leak.Fit(xs, ys)

	if math.Abs(slope-10) > 1e-9 {
		t.Errorf("slope = %v, want 10", slope)
	}
	if math.Abs(r2-1) > 1e-9 {
		t.Errorf("r2 = %v, want 1", r2)
	}
}

func TestFitFlatSeriesHasZeroSlope(t *testing.T) {
	t.Parallel()

	slope, r2 := leak.Fit([]float64{0, 1, 2}, []int64{5, 5, 5})

	if slope != 0 {
		t.Errorf("slope = %v, want 0", slope)
	}
	if r2 != 0 {
		t.Errorf("r2 = %v, want 0 (no variance to explain)", r2)
	}
}

func TestFitTooFewPoints(t *testing.T) {
	t.Parallel()

	slope, r2 := leak.Fit([]float64{0}, []int64{42})

	if slope != 0 || r2 != 0 {
		t.Errorf("fit() = (%v, %v), want (0, 0)", slope, r2)
	}
}

func TestRankSurfacesTheLeakFirst(t *testing.T) {
	t.Parallel()

	leakSite := profile.Frame{Function: "(*Cache).Set", File: "cache/store.go", Line: 42}
	stable := profile.Frame{Function: "(*Pool).Get", File: "pool/pool.go", Line: 10}

	snaps := []leak.Snapshot{
		snapshot(0, alloc(leakSite, 10), alloc(stable, 100)),
		snapshot(1, alloc(leakSite, 20), alloc(stable, 100)),
		snapshot(2, alloc(leakSite, 30), alloc(stable, 100)),
		snapshot(3, alloc(leakSite, 40), alloc(stable, 100)),
	}

	got := leak.Rank(snaps)

	if got[0].Function != "(*Cache).Set" {
		t.Fatalf("top growth = %q, want (*Cache).Set\n%+v", got[0].Function, got)
	}
	if math.Abs(got[0].Slope-10) > 1e-9 {
		t.Errorf("leak slope = %v, want 10", got[0].Slope)
	}
	if got[0].R2 < 0.99 {
		t.Errorf("leak R2 = %v, want ~1", got[0].R2)
	}
	if got[0].Released {
		t.Error("leak Released = true, want false (it never drops)")
	}
	if got[0].First != 10 || got[0].Last != 40 {
		t.Errorf("leak First/Last = %d/%d, want 10/40", got[0].First, got[0].Last)
	}

	s := find(t, got, "(*Pool).Get")
	if s.Slope != 0 {
		t.Errorf("stable slope = %v, want 0", s.Slope)
	}
}

func TestRankFlagsReleasedSites(t *testing.T) {
	t.Parallel()

	sawtooth := profile.Frame{Function: "(*Buffer).Grow", File: "buf/buf.go", Line: 7}

	snaps := []leak.Snapshot{
		snapshot(0, alloc(sawtooth, 10)),
		snapshot(1, alloc(sawtooth, 40)),
		snapshot(2, alloc(sawtooth, 20)), // dropped: memory was released here
		snapshot(3, alloc(sawtooth, 50)),
	}

	g := find(t, leak.Rank(snaps), "(*Buffer).Grow")
	if !g.Released {
		t.Error("Released = false, want true (bytes dropped between snapshots)")
	}
}
