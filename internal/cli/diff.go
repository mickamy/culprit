package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/mickamy/culprit/internal/diff"
	"github.com/mickamy/culprit/internal/exit"
	"github.com/mickamy/culprit/internal/profile"
	"github.com/mickamy/culprit/internal/render"
)

// runDiff runs the diff engine on any two pprof profiles, ranking the
// per-function flat delta and pointing at the line that moved most.
func runDiff(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	sampleType := fs.String("type", "", "sample type to diff (defaults to the profile's own)")
	top := fs.Int("top", 10, "show the top n movers per section")

	if err := fs.Parse(args); err != nil {
		return exit.Usage
	}

	files := fs.Args()
	if len(files) != 2 {
		fmt.Fprintln(stderr, "culprit diff: need two profiles: <base> <head>")

		return exit.Usage
	}

	base, err := profile.LoadFile(files[0])
	if err != nil {
		fmt.Fprintf(stderr, "culprit diff: %v\n", err)

		return exit.Error
	}

	head, err := profile.LoadFile(files[1])
	if err != nil {
		fmt.Fprintf(stderr, "culprit diff: %v\n", err)

		return exit.Error
	}

	st := *sampleType
	if st == "" {
		st = profile.DefaultSampleType(head)
	}

	baseSamples, err := profile.Samples(base, st)
	if err != nil {
		fmt.Fprintf(stderr, "culprit diff: %v\n", err)

		return exit.Error
	}

	headSamples, err := profile.Samples(head, st)
	if err != nil {
		fmt.Fprintf(stderr, "culprit diff: %v\n", err)

		return exit.Error
	}

	render.Diff(stdout, diff.Rank(baseSamples, headSamples), render.DiffReport{
		SampleType: st,
		Unit:       profile.Unit(head, st),
		Top:        *top,
	})

	return exit.OK
}

func printDiffUsage(w io.Writer) {
	fmt.Fprintln(w, "culprit diff — diff two pprof profiles into a culprit ranking.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE:")
	fmt.Fprintln(w, "  culprit diff <base.prof> <head.prof>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Ranks functions by their flat delta (own cost) between any two pprof profiles")
	fmt.Fprintln(w, "and points at the line that moved most. Works with CPU, heap, or any pprof profile.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "FLAGS:")
	fmt.Fprintln(w, "  --type <name>    Sample type to diff (default: the profile's own)")
	fmt.Fprintln(w, "  --top <n>        Show the top n movers per section")
	fmt.Fprintln(w, "  --help, -h       Show this help")
}
