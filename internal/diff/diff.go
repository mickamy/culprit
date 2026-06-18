package diff

import (
	"cmp"
	"slices"

	"github.com/mickamy/culprit/internal/profile"
)

type Change struct {
	profile.Frame

	BaseFlat int64
	HeadFlat int64
	BaseCum  int64
	HeadCum  int64
}

func (c Change) FlatDelta() int64 { return c.HeadFlat - c.BaseFlat }

func (c Change) CumDelta() int64 { return c.HeadCum - c.BaseCum }

// Rank diffs base against head per function and orders the result by the
// magnitude of each function's flat delta — the code whose own cost moved most.
// A positive FlatDelta is a regression, a negative one an improvement.
func Rank(base, head []profile.Sample) []Change {
	b := aggregate(base)
	h := aggregate(head)

	changes := make([]Change, 0, len(b)+len(h))

	for k := range union(b, h) {
		changes = append(changes, Change{
			Frame:    profile.Frame{Function: k.function, File: k.file, Line: dominantLine(h, b, k)},
			BaseFlat: b.flat(k),
			HeadFlat: h.flat(k),
			BaseCum:  b.cum(k),
			HeadCum:  h.cum(k),
		})
	}

	slices.SortStableFunc(changes, func(a, b Change) int {
		if d := cmp.Compare(abs(b.FlatDelta()), abs(a.FlatDelta())); d != 0 {
			return d
		}
		if d := cmp.Compare(a.Function, b.Function); d != 0 {
			return d
		}

		return cmp.Compare(a.File, b.File)
	})

	return changes
}

type key struct {
	function string
	file     string
}

type lineKey struct {
	key

	line int64
}

type bucket struct {
	flat  int64
	cum   int64
	lines map[int64]int64 // cum-weighted contribution per line, for localization
}

// aggregate sums flat and cumulative values per function. Flat is attributed to
// the leaf frame; cumulative counts a sample once per distinct function it
// touches, so recursion does not inflate it.
func aggregate(samples []profile.Sample) buckets {
	acc := make(buckets)
	seen := make(map[key]bool)
	seenLine := make(map[lineKey]bool)

	for _, s := range samples {
		if len(s.Stack) == 0 {
			continue
		}

		leaf := s.Stack[0]
		acc.at(key{leaf.Function, leaf.File}).flat += s.Value

		clear(seen)
		clear(seenLine)

		for _, f := range s.Stack {
			k := key{f.Function, f.File}
			b := acc.at(k)

			if !seen[k] {
				seen[k] = true
				b.cum += s.Value
			}

			lk := lineKey{k, f.Line}
			if !seenLine[lk] {
				seenLine[lk] = true
				b.lines[f.Line] += s.Value
			}
		}
	}

	return acc
}

type buckets map[key]*bucket

func (bs buckets) at(k key) *bucket {
	b := bs[k]
	if b == nil {
		b = &bucket{lines: make(map[int64]int64)}
		bs[k] = b
	}

	return b
}

func (bs buckets) flat(k key) int64 {
	if b := bs[k]; b != nil {
		return b.flat
	}

	return 0
}

func (bs buckets) cum(k key) int64 {
	if b := bs[k]; b != nil {
		return b.cum
	}

	return 0
}

func (b *bucket) hottestLine() int64 {
	if b == nil {
		return 0
	}

	var best, bestVal int64

	for line, v := range b.lines {
		if v > bestVal || (v == bestVal && line < best) {
			best, bestVal = line, v
		}
	}

	return best
}

func union(a, b buckets) map[key]struct{} {
	keys := make(map[key]struct{}, len(a)+len(b))
	for k := range a {
		keys[k] = struct{}{}
	}
	for k := range b {
		keys[k] = struct{}{}
	}

	return keys
}

// dominantLine returns the line carrying the most cumulative value, preferring
// head and falling back to base for functions that only exist in one side.
func dominantLine(head, base buckets, k key) int64 {
	if l := head[k].hottestLine(); l != 0 {
		return l
	}

	return base[k].hottestLine()
}

func abs(n int64) uint64 {
	if n < 0 {
		return uint64(-n)
	}

	return uint64(n)
}
