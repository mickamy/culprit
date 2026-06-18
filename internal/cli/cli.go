package cli

import (
	"fmt"
	"io"

	"github.com/mickamy/culprit/internal/exit"
)

type subcommand struct {
	name    string
	summary string
	run     func(args []string, stdout, stderr io.Writer) int
	usage   func(w io.Writer)
}

var subcommands = []subcommand{
	{
		name:    "bench",
		summary: "Run benchmarks on HEAD vs a base ref, diff profiles, name regressions",
		run:     runBench,
		usage:   printBenchUsage,
	},
	{
		name:    "leak",
		summary: "Sample heap over a run, find the allocation site that keeps growing",
		run:     runLeak,
		usage:   printLeakUsage,
	},
	{
		name:    "diff",
		summary: "Diff two pprof profiles into a ranked culprit list",
		run:     runDiff,
		usage:   printDiffUsage,
	},
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		PrintUsage(stderr)

		return exit.Usage
	}

	c, ok := lookup(args[0])
	if !ok {
		fmt.Fprintf(stderr, "culprit: unknown command %q\n", args[0])
		fmt.Fprintln(stderr, "Run 'culprit --help' for usage.")

		return exit.Usage
	}

	rest := args[1:]
	if wantsHelp(rest) {
		c.usage(stdout)

		return exit.OK
	}

	return c.run(rest, stdout, stderr)
}

func lookup(name string) (subcommand, bool) {
	for _, c := range subcommands {
		if c.name == name {
			return c, true
		}
	}

	return subcommand{}, false
}

func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}

	return false
}

// runBench builds HEAD and --base in separate git worktrees, runs go test
// -bench with CPU and memory profiling on each, diffs the profiles, and ranks
// the per-function cumulative delta. The top mover is the regression's culprit.
func runBench(args []string, stdout, stderr io.Writer) int {
	return notImplemented(args, stdout, stderr)
}

// runLeak samples a process's heap over time — either by starting it with
// -- <cmd> or by polling an existing --pprof endpoint — builds a time series of
// inuse_space per allocation site, and ranks the sites that keep growing.
func runLeak(args []string, stdout, stderr io.Writer) int {
	return notImplemented(args, stdout, stderr)
}

func PrintUsage(w io.Writer) {
	fmt.Fprintln(w, "culprit — find the culprit behind a Go slowdown or memory leak, and name the line.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE:")
	fmt.Fprintln(w, "  culprit <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "COMMANDS:")

	width := 0

	for _, c := range subcommands {
		width = max(width, len(c.name))
	}

	for _, c := range subcommands {
		fmt.Fprintf(w, "  %-*s  %s\n", width, c.name, c.summary)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "EXAMPLES:")
	fmt.Fprintln(w, "  # Diff HEAD against main and name the regressions")
	fmt.Fprintln(w, "  culprit bench --base main")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  # Watch a running worker's heap and find the growing allocation site")
	fmt.Fprintln(w, "  culprit leak -- ./bin/worker")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  # Diff two pprof profiles you already have")
	fmt.Fprintln(w, "  culprit diff base.prof head.prof")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "FLAGS:")
	fmt.Fprintln(w, "  --version, -v    Print culprit version")
	fmt.Fprintln(w, "  --help, -h       Show this help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'culprit <command> --help' for command-specific flags.")
	fmt.Fprintln(w, "More: https://github.com/mickamy/culprit")
}

func printBenchUsage(w io.Writer) {
	fmt.Fprintln(w, "culprit bench — run benchmarks on HEAD vs a base ref and name regressions.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE:")
	fmt.Fprintln(w, "  culprit bench [--base <ref>] [--bench <re>] [pkg]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Builds HEAD and <ref> in separate git worktrees, runs 'go test -bench' with CPU")
	fmt.Fprintln(w, "and memory profiling on each, then ranks the per-function cumulative delta.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "FLAGS:")
	fmt.Fprintln(w, "  --base <ref>     Compare HEAD against this ref (default: main)")
	fmt.Fprintln(w, "  --bench <re>     Only run benchmarks matching this regexp (default: all)")
	fmt.Fprintln(w, "  --count <n>      Run each benchmark n times for noise reduction")
	fmt.Fprintln(w, "  --flame <file>   Write a differential flamegraph SVG here")
	fmt.Fprintln(w, "  --top <n>        Show the top n movers")
	fmt.Fprintln(w, "  -o <file>        Write the report here instead of the terminal")
	fmt.Fprintln(w, "  --help, -h       Show this help")
}

func printLeakUsage(w io.Writer) {
	fmt.Fprintln(w, "culprit leak — sample a running process's heap and find the growing site.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE:")
	fmt.Fprintln(w, "  culprit leak -- <cmd> [args...]")
	fmt.Fprintln(w, "  culprit leak --pprof <url>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Captures heap profiles every few seconds, builds a time series of inuse_space")
	fmt.Fprintln(w, "per allocation site, and ranks the sites that keep growing. Stop with ctrl-c.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "FLAGS:")
	fmt.Fprintln(w, "  --pprof <url>    Sample an already-running process at this pprof endpoint")
	fmt.Fprintln(w, "  --interval <d>   Time between heap samples (e.g., 2s)")
	fmt.Fprintln(w, "  --flame <file>   Write a differential flamegraph SVG here")
	fmt.Fprintln(w, "  --top <n>        Show the top n growing sites")
	fmt.Fprintln(w, "  -o <file>        Write the report here instead of the terminal")
	fmt.Fprintln(w, "  --help, -h       Show this help")
}

func notImplemented(_ []string, _, stderr io.Writer) int {
	fmt.Fprintln(stderr, "culprit: not implemented yet")

	return exit.NotImplemented
}
