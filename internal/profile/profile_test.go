package profile_test

import (
	"strings"
	"testing"

	pprof "github.com/google/pprof/profile"

	"github.com/mickamy/culprit/internal/profile"
)

func sampleProfile() *pprof.Profile {
	enrich := &pprof.Function{ID: 1, Name: "(*OrderUseCase).enrich", Filename: "internal/usecase/order.go"}
	marshal := &pprof.Function{ID: 2, Name: "json.Marshal", Filename: "encoding/json/encode.go"}
	mainFn := &pprof.Function{ID: 3, Name: "main.main", Filename: "main.go"}

	// marshal is inlined into enrich, so the leaf location carries both lines
	// leaf-first: the innermost callee first, its caller last.
	leaf := &pprof.Location{
		ID: 1,
		Line: []pprof.Line{
			{Function: marshal, Line: 200},
			{Function: enrich, Line: 88},
		},
	}
	main := &pprof.Location{ID: 2, Line: []pprof.Line{{Function: mainFn, Line: 10}}}
	stripped := &pprof.Location{ID: 3, Address: 0xdeadbeef}

	return &pprof.Profile{
		SampleType: []*pprof.ValueType{
			{Type: "count", Unit: "count"},
			{Type: "cpu", Unit: "nanoseconds"},
		},
		Function: []*pprof.Function{enrich, marshal, mainFn},
		Location: []*pprof.Location{leaf, main, stripped},
		Sample: []*pprof.Sample{
			{Location: []*pprof.Location{leaf, main}, Value: []int64{1, 100}},
			{Location: []*pprof.Location{stripped}, Value: []int64{1, 5}},
		},
	}
}

func TestSamplesExpandsInlinedFramesLeafFirst(t *testing.T) {
	t.Parallel()

	got, err := profile.Samples(sampleProfile(), "cpu")
	if err != nil {
		t.Fatalf("Samples() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d samples, want 2", len(got))
	}

	want := []profile.Frame{
		{Function: "json.Marshal", File: "encoding/json/encode.go", Line: 200},
		{Function: "(*OrderUseCase).enrich", File: "internal/usecase/order.go", Line: 88},
		{Function: "main.main", File: "main.go", Line: 10},
	}

	if len(got[0].Stack) != len(want) {
		t.Fatalf("stack = %+v, want %d frames", got[0].Stack, len(want))
	}
	for i, f := range want {
		if got[0].Stack[i] != f {
			t.Errorf("frame[%d] = %+v, want %+v", i, got[0].Stack[i], f)
		}
	}
	if got[0].Value != 100 {
		t.Errorf("value = %d, want 100 (cpu, not count)", got[0].Value)
	}
}

func TestSamplesUnsymbolizedLocation(t *testing.T) {
	t.Parallel()

	got, err := profile.Samples(sampleProfile(), "cpu")
	if err != nil {
		t.Fatalf("Samples() error = %v", err)
	}

	stack := got[1].Stack
	if len(stack) != 1 {
		t.Fatalf("stack = %+v, want 1 frame", stack)
	}
	if stack[0].Function != "0xdeadbeef" {
		t.Errorf("function = %q, want address fallback %q", stack[0].Function, "0xdeadbeef")
	}
}

func TestSamplesUnknownType(t *testing.T) {
	t.Parallel()

	_, err := profile.Samples(sampleProfile(), "inuse_space")
	if err == nil {
		t.Fatal("Samples() error = nil, want error for unknown sample type")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention the missing type", err)
	}
}

func TestSampleTypes(t *testing.T) {
	t.Parallel()

	got := profile.SampleTypes(sampleProfile())
	want := []string{"count", "cpu"}

	if len(got) != len(want) {
		t.Fatalf("SampleTypes() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SampleTypes()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDefaultSampleType(t *testing.T) {
	t.Parallel()

	if got := profile.DefaultSampleType(sampleProfile()); got != "cpu" {
		t.Errorf("DefaultSampleType() = %q, want %q (last type when unset)", got, "cpu")
	}

	p := sampleProfile()
	p.DefaultSampleType = "count"
	if got := profile.DefaultSampleType(p); got != "count" {
		t.Errorf("DefaultSampleType() = %q, want %q", got, "count")
	}
}
