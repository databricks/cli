package pipelines

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// testLineWidth is the width to which a case line is padded with dots before its status. The
// fixed width lines up the statuses while each case prints as it finishes.
const testLineWidth = 80

// testProgressPrinter prints the phase of a test update and each case as it finishes.
type testProgressPrinter struct {
	out      io.Writer
	shortID  func(id string) string
	finished int
	waiting  bool
	running  bool
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

// caseFinished prints a line such as
// "2 of 3 FAILED tests/test_orders.py::test_backfill ...... [FAILED in 2.14s]". total is the
// number of known cases; the service reports every case as PENDING before the first one runs.
func (p *testProgressPrinter) caseFinished(c *testCaseResult, total int) {
	p.startRunning()
	p.finished++
	prefix := fmt.Sprintf("%d of %d %s %s ", p.finished, total, c.status, p.shortID(c.id))
	status := string(c.status)
	if d, ok := c.duration(); ok {
		status += fmt.Sprintf(" in %.2fs", d.Seconds())
	}
	dots := strings.Repeat(".", max(testLineWidth-utf8.RuneCountInString(prefix), 0))
	fmt.Fprintf(p.out, "%s%s [%s]\n", prefix, dots, status)
}

// renderTestResult prints the end of a test run, before its summary line: the details of every
// failed or errored case, then why the result is incomplete or that no tests were collected.
func renderTestResult(out io.Writer, run *testRun, incompleteReason string, shortID func(id string) string) {
	fmt.Fprintln(out)
	counts := run.counts()
	if counts.Failed > 0 || counts.Errored > 0 {
		fmt.Fprintf(out, "Completed with %d failed and %d errored tests:\n", counts.Failed, counts.Errored)
		for _, c := range run.cases {
			var label string
			switch c.status {
			case testCaseFailed:
				label = "Failure in"
			case testCaseError:
				label = "Error in"
			default:
				continue
			}
			fmt.Fprintf(out, "\n%s %s\n", label, shortID(c.id))
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
		fmt.Fprintln(out)
	}

	switch {
	case incompleteReason != "":
		fmt.Fprintf(out, "RUN INCOMPLETE: %s.\n", incompleteReason)
	case run.summaries[0].Total == 0:
		fmt.Fprintln(out, "No tests were collected.")
	}
}

// renderTestSummaryLine prints the last line of a test run, "Done. PASSED=… TOTAL=…".
// A complete run shows the service's summary. An incomplete run shows the results received, and
// its total counts every case the events named.
func renderTestSummaryLine(out io.Writer, run *testRun, incompleteReason string) {
	s := run.counts()
	s.Total = len(run.cases)
	if incompleteReason == "" {
		s = run.summaries[0]
	}
	fmt.Fprintf(out, "Done. PASSED=%d FAILED=%d SKIPPED=%d ERROR=%d TOTAL=%d\n", s.Passed, s.Failed, s.Skipped, s.Errored, s.Total)
}
