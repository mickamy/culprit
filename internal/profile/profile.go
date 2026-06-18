package profile

import (
	"fmt"
	"io"
	"strings"

	"github.com/google/pprof/profile"
)

type Frame struct {
	Function string
	File     string
	Line     int64
}

type Sample struct {
	Stack []Frame
	Value int64
}

func Load(r io.Reader) (*profile.Profile, error) {
	p, err := profile.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}

	return p, nil
}

// Samples normalizes p into leaf-first stacks carrying the value of the named
// sample type (e.g., "cpu", "inuse_space", "alloc_space"). Inlined frames are
// expanded so each resolves to its own file:line.
func Samples(p *profile.Profile, sampleType string) ([]Sample, error) {
	idx, err := valueIndex(p, sampleType)
	if err != nil {
		return nil, err
	}

	samples := make([]Sample, 0, len(p.Sample))

	for _, s := range p.Sample {
		stack := make([]Frame, 0, len(s.Location))

		for _, loc := range s.Location {
			if len(loc.Line) == 0 {
				stack = append(stack, Frame{Function: fmt.Sprintf("0x%x", loc.Address)})

				continue
			}

			for _, line := range loc.Line {
				stack = append(stack, frameOf(line))
			}
		}

		samples = append(samples, Sample{Stack: stack, Value: s.Value[idx]})
	}

	return samples, nil
}

func SampleTypes(p *profile.Profile) []string {
	types := make([]string, len(p.SampleType))
	for i, vt := range p.SampleType {
		types[i] = vt.Type
	}

	return types
}

// DefaultSampleType reports the profile's preferred sample type, falling back to
// the last type (pprof's own default) when the profile leaves it unset.
func DefaultSampleType(p *profile.Profile) string {
	if p.DefaultSampleType != "" {
		return p.DefaultSampleType
	}

	if len(p.SampleType) == 0 {
		return ""
	}

	return p.SampleType[len(p.SampleType)-1].Type
}

func frameOf(line profile.Line) Frame {
	f := Frame{Line: line.Line}
	if line.Function != nil {
		f.Function = line.Function.Name
		f.File = line.Function.Filename
	}

	return f
}

func valueIndex(p *profile.Profile, sampleType string) (int, error) {
	for i, vt := range p.SampleType {
		if vt.Type == sampleType {
			return i, nil
		}
	}

	return 0, fmt.Errorf("sample type %q not found (have: %s)", sampleType, strings.Join(SampleTypes(p), ", "))
}
