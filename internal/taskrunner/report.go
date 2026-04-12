package taskrunner

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// PrintReport renders the final report to the writer.
func PrintReport(w io.Writer, report *Report) {
	divider := strings.Repeat("─", 60)

	// Header
	fmt.Fprintln(w, divider)
	if report.Status == "success" {
		fmt.Fprintln(w, "  ✓ DONE")
	} else {
		fmt.Fprintln(w, "  ✗ FAILED")
	}
	fmt.Fprintln(w, divider)

	// Metadata
	fmt.Fprintf(w, "  Task:       %s\n", report.Task)
	fmt.Fprintf(w, "  Iterations: %d\n", report.Iterations)
	fmt.Fprintf(w, "  Duration:   %s\n", report.Duration.Round(time.Second))
	fmt.Fprintf(w, "  Engine:     %s\n", report.EngineUsed)
	fmt.Fprintf(w, "  WorkDir:    %s\n", report.WorkDir)
	fmt.Fprintln(w, divider)

	// Summary
	fmt.Fprintln(w, "  Summary:")
	fmt.Fprintln(w)
	for _, line := range strings.Split(report.FinalOutput, "\n") {
		fmt.Fprintf(w, "    %s\n", line)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, divider)
}

// PrintVerboseReport includes step details.
func PrintVerboseReport(w io.Writer, report *Report) {
	PrintReport(w, report)

	if len(report.Steps) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  Steps:")
		for _, step := range report.Steps {
			status := "✓"
			if step.Error != "" {
				status = "✗"
			}
			fmt.Fprintf(w, "    %s Step %d: %s (%s)\n", status, step.Iteration, step.Action.Type, step.Duration.Round(time.Millisecond))
		}
		fmt.Fprintln(w, strings.Repeat("─", 60))
	}
}
