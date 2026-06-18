package leak

import (
	"sort"

	"github.com/mickamy/culprit/internal/profile"
)

// Snapshot is one heap profile captured at a point in time. Seconds is the
// elapsed time since the first snapshot; Samples carry inuse_space per site.
type Snapshot struct {
	Seconds float64
	Samples []profile.Sample
}

type Growth struct {
	profile.Frame

	Slope    float64 // retained bytes per second, from the least-squares fit
	R2       float64 // goodness of fit, 0..1
	First    int64
	Last     int64
	Released bool // whether retained bytes ever dropped between snapshots
}

// Rank builds a per-site time series of retained bytes across the snapshots and
// orders sites by growth slope. A site that climbs steadily and is never
// released — high Slope, high R2, Released false — is the leak.
func Rank(snapshots []Snapshot) []Growth {
	xs := make([]float64, len(snapshots))
	perSnapshot := make([]map[profile.Frame]int64, len(snapshots))
	sites := make(map[profile.Frame]struct{})

	for i, snap := range snapshots {
		xs[i] = snap.Seconds
		flat := flatBySite(snap.Samples)
		perSnapshot[i] = flat

		for site := range flat {
			sites[site] = struct{}{}
		}
	}

	growths := make([]Growth, 0, len(sites))

	for site := range sites {
		ys := make([]int64, len(snapshots))
		for i := range snapshots {
			ys[i] = perSnapshot[i][site]
		}

		growths = append(growths, growthOf(site, xs, ys))
	}

	sort.SliceStable(growths, func(i, j int) bool {
		a, b := growths[i], growths[j]
		if a.Slope != b.Slope {
			return a.Slope > b.Slope
		}
		if a.Last != b.Last {
			return a.Last > b.Last
		}

		return a.Function < b.Function
	})

	return growths
}

func flatBySite(samples []profile.Sample) map[profile.Frame]int64 {
	flat := make(map[profile.Frame]int64)

	for _, s := range samples {
		if len(s.Stack) == 0 {
			continue
		}

		flat[s.Stack[0]] += s.Value
	}

	return flat
}

func growthOf(site profile.Frame, xs []float64, ys []int64) Growth {
	slope, r2 := fit(xs, ys)

	g := Growth{Frame: site, Slope: slope, R2: r2}
	if len(ys) > 0 {
		g.First = ys[0]
		g.Last = ys[len(ys)-1]
	}

	for i := 1; i < len(ys); i++ {
		if ys[i] < ys[i-1] {
			g.Released = true

			break
		}
	}

	return g
}

// fit computes the least-squares regression of ys over xs, returning the slope
// and the R2 goodness of fit. It returns zeros when there are too few points or
// no variance in x to fit a line through.
func fit(xs []float64, ys []int64) (slope, r2 float64) {
	n := len(xs)
	if n < 2 {
		return 0, 0
	}

	var sumX, sumY float64
	for i := range xs {
		sumX += xs[i]
		sumY += float64(ys[i])
	}

	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	var sxy, sxx, syy float64
	for i := range xs {
		dx := xs[i] - meanX
		dy := float64(ys[i]) - meanY
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}

	if sxx == 0 {
		return 0, 0
	}

	slope = sxy / sxx
	if syy == 0 {
		return slope, 0
	}

	r2 = (sxy * sxy) / (sxx * syy)

	return slope, r2
}
