package pipelines

import (
	"fmt"
	"io"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// testProgressPrinter prints the phase of a test update and each case as it finishes.
type testProgressPrinter struct {
	out     io.Writer
	waiting bool
	running bool
}

func (p *testProgressPrinter) updateState(state pipelines.UpdateInfoState) {
	switch state {
	case pipelines.UpdateInfoStateQueued,
		pipelines.UpdateInfoStateCreated,
		pipelines.UpdateInfoStateWaitingForResources,
		pipelines.UpdateInfoStateInitializing,
		pipelines.UpdateInfoStateResetting,
		pipelines.UpdateInfoStateSettingUpTables:
		if !p.waiting && !p.running {
			fmt.Fprintln(p.out, "Waiting for compute...")
			p.waiting = true
		}
	case pipelines.UpdateInfoStateRunning:
		p.startRunning()
	default:
	}
}

func (p *testProgressPrinter) startRunning() {
	if !p.running {
		fmt.Fprintln(p.out, "Running tests...")
		p.running = true
	}
}

func (p *testProgressPrinter) caseFinished(c *testCaseResult) {
	p.startRunning()
	if d, ok := c.duration(); ok {
		fmt.Fprintf(p.out, "%s  %s (%.2fs)\n", c.name, c.status, d.Seconds())
	} else {
		fmt.Fprintf(p.out, "%s  %s\n", c.name, c.status)
	}
}

// renderTestFailures prints the details of every failed or errored case.
func renderTestFailures(out io.Writer, run *testRun) {
	var failed []*testCaseResult
	for _, c := range run.cases {
		if c.status == testCaseFailed || c.status == testCaseError {
			failed = append(failed, c)
		}
	}
	if len(failed) == 0 {
		return
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "FAILURES")
	for _, c := range failed {
		fmt.Fprintln(out, c.name)
		details := ""
		if c.errorDetail != nil {
			details = errorDetailText(c.errorDetail)
		}
		if details == "" {
			details = "No details were reported."
		}
		for line := range strings.SplitSeq(details, "\n") {
			fmt.Fprintf(out, "  %s\n", line)
		}
		if c.detailsTruncated {
			fmt.Fprintln(out, "  (details truncated)")
		}
	}
}

func errorsLabel(n int) string {
	if n == 1 {
		return "error"
	}
	return "errors"
}

func formatTestCounts(s testRunSummary) string {
	return fmt.Sprintf("%d passed, %d failed, %d skipped, %d %s", s.Passed, s.Failed, s.Skipped, s.Errors, errorsLabel(s.Errors))
}

// renderTestResult prints the end of a test run: failure details, then the summary or why
// the result is incomplete.
func renderTestResult(out io.Writer, run *testRun, incompleteReason string) {
	renderTestFailures(out, run)
	fmt.Fprintln(out)

	if incompleteReason != "" {
		fmt.Fprintf(out, "RUN INCOMPLETE: %s.\n", incompleteReason)
		received := run.counts()
		var parts []string
		for _, count := range []struct {
			n     int
			label string
		}{
			{received.Passed, "passed"},
			{received.Failed, "failed"},
			{received.Skipped, "skipped"},
			{received.Errors, errorsLabel(received.Errors)},
		} {
			if count.n > 0 {
				parts = append(parts, fmt.Sprintf("%d %s", count.n, count.label))
			}
		}
		if len(parts) == 0 {
			parts = []string{"none"}
		}
		fmt.Fprintf(out, "Results received: %s.\n", strings.Join(parts, ", "))
		return
	}

	summary := run.summaries[0]
	switch summary.Total {
	case 0:
		fmt.Fprintln(out, "No tests were collected.")
	case 1:
		fmt.Fprintf(out, "1 test: %s\n", formatTestCounts(summary))
	default:
		fmt.Fprintf(out, "%d tests: %s\n", summary.Total, formatTestCounts(summary))
	}
}
