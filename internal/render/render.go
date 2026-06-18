package render

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/mickamy/culprit/internal/diff"
)

type DiffReport struct {
	SampleType string
	Unit       string
	Top        int
}

// Diff writes a ranked culprit report: regressions then improvements, each
// localized to function and file:line, with the flat delta humanized by unit.
func Diff(w io.Writer, changes []diff.Change, report DiffReport) {
	fmt.Fprintf(w, "diff: %s (%s)\n\n", report.SampleType, report.Unit)

	var regressions, improvements []diff.Change

	for _, c := range changes {
		switch {
		case c.FlatDelta() > 0:
			regressions = append(regressions, c)
		case c.FlatDelta() < 0:
			improvements = append(improvements, c)
		}
	}

	if len(regressions) == 0 && len(improvements) == 0 {
		fmt.Fprintln(w, "no change")

		return
	}

	section(w, "REGRESSIONS", regressions, report)
	section(w, "IMPROVEMENTS", improvements, report)
}

func section(w io.Writer, title string, changes []diff.Change, report DiffReport) {
	if len(changes) == 0 {
		return
	}

	fmt.Fprintf(w, "%s (%d)\n", title, len(changes))

	shown := changes
	truncated := 0

	if report.Top > 0 && len(changes) > report.Top {
		shown = changes[:report.Top]
		truncated = len(changes) - report.Top
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, c := range shown {
		fmt.Fprintf(tw, "  %s\t%s\t%s:%d\n", humanize(c.FlatDelta(), report.Unit), c.Function, c.File, c.Line)
	}
	_ = tw.Flush()

	if truncated > 0 {
		fmt.Fprintf(w, "  … and %d more\n", truncated)
	}

	fmt.Fprintln(w)
}

func humanize(v int64, unit string) string {
	sign := "+"
	mag := v

	if v < 0 {
		sign = "-"
		mag = -v
	}

	switch unit {
	case "nanoseconds":
		return sign + time.Duration(mag).String()
	case "bytes":
		return sign + formatBytes(mag)
	default:
		return fmt.Sprintf("%+d", v)
	}
}

func formatBytes(b int64) string {
	const unit = 1024

	if b < unit {
		return fmt.Sprintf("%dB", b)
	}

	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
