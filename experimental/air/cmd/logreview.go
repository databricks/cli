package aircmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

const (
	// reviewChunksFromEnd is how many newest chunks per node the review fetches.
	// Error signatures live in the most recent activity, so a couple of chunks is
	// enough and avoids downloading a long run's entire history.
	reviewChunksFromEnd = 2
	// defaultReviewLines is how many trailing lines per node are analyzed when
	// --lines is not given.
	defaultReviewLines = 200
)

// reviewErrorKeywords mark a log line as interesting when reviewing a run. They
// are matched case-insensitively against each line.
var reviewErrorKeywords = []string{"error", "failed", "timeout", "exception"}

// isReviewErrorLine reports whether a line looks like a failure worth
// highlighting in the review output.
func isReviewErrorLine(line string) bool {
	lower := strings.ToLower(line)
	for _, kw := range reviewErrorKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// reviewLines is the number of trailing lines per node the review analyzes:
// --lines when set, else the default.
func (req logRequest) reviewLines() int {
	if req.tailLines > 0 {
		return req.tailLines
	}
	return defaultReviewLines
}

// reviewLogs downloads every node's recent logs to a temp directory and prints
// each node's tail with failure lines highlighted, so a failed multi-node run can
// be triaged in one command. Returns whether the run succeeded, matching the
// other log paths' exit-code behavior.
func reviewLogs(ctx context.Context, w *databricks.WorkspaceClient, out io.Writer, req logRequest, status logRunStatus) (bool, error) {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: req.runID})
	if err != nil {
		return false, err
	}

	numNodes, err := resolveNodeCount(run)
	if err != nil {
		return false, err
	}

	ids := mlflowIDs(ctx, w, run)
	if ids == nil || ids.RunID == "" {
		return status.succeeded(), fmt.Errorf("no logs available for run %d", req.runID)
	}

	// Review output is transient, so stage the downloads in a temp dir.
	dir, err := os.MkdirTemp("", "air-review-*")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(dir)

	nodes := make([]int, 0, numNodes)
	for n := range numNodes {
		nodes = append(nodes, n)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Reviewing logs from %d node(s)...", numNodes))
	nodeLogs, err := downloadAllNodeLogs(ctx, w, ids.RunID, dir, nodes, req.attempt, reviewChunksFromEnd)
	if err != nil {
		return false, err
	}
	if len(nodeLogs) == 0 {
		return status.succeeded(), fmt.Errorf("no logs available for run %d", req.runID)
	}

	renderReview(ctx, out, nodeLogs, req.reviewLines())
	return status.succeeded(), nil
}

// renderReview prints each node's trailing lines in a box, highlighting lines
// that match a failure keyword.
func renderReview(ctx context.Context, out io.Writer, nodeLogs map[int]string, lines int) {
	renderer, _ := cmdio.NewRenderer(ctx, out)
	p := newPalette(renderer)

	for _, node := range sortedNodeKeys(nodeLogs) {
		tail, err := tailFileLines(nodeLogs[node], lines)
		if err != nil || len(tail) == 0 {
			continue
		}

		styled := make([]string, 0, len(tail))
		for _, line := range tail {
			if isReviewErrorLine(line) {
				styled = append(styled, p.red.Render(line))
			} else {
				styled = append(styled, p.n12.Render(line))
			}
		}

		title := fmt.Sprintf("Node %d — last %d lines", node, len(tail))
		fmt.Fprintf(out, "\n%s\n", renderBox(p, title, strings.Join(styled, "\n")))
	}
}

// tailFileLines returns the last n lines of a file.
func tailFileLines(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Ring buffer so a large log doesn't need to be held in memory at once.
	buf := make([]string, 0, n)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		if len(buf) == n {
			buf = buf[1:]
		}
		buf = append(buf, scanner.Text())
	}
	return buf, scanner.Err()
}
