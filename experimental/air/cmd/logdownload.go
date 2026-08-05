package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"golang.org/x/sync/errgroup"
)

// downloadConcurrency caps how many nodes download at once, to avoid hammering
// the artifact store on a wide run.
const downloadConcurrency = 8

// errNodeOutOfRange marks --node naming a node the run doesn't have. It is user
// input, so the caller reports it as an invalid argument rather than a failure.
var errNodeOutOfRange = errors.New("invalid --node")

// resolveNodeCount returns how many nodes a run used.
func resolveNodeCount(run *jobs.Run) (int, error) {
	accelType, count := jobCompute(run)
	if accelType == "" {
		return 0, fmt.Errorf("run %d has no AI runtime compute config", run.RunId)
	}
	if count <= 0 {
		return 0, fmt.Errorf("run %d reports %d accelerators", run.RunId, count)
	}
	g, err := parseGPUType(accelType)
	if err != nil {
		return 0, err
	}
	perNode, err := gpusPerNode(g)
	if err != nil {
		return 0, err
	}
	// Accelerators come in whole nodes, so a remainder means we can't map the
	// count onto node indices.
	if count%perNode != 0 {
		return 0, fmt.Errorf("run %d reports %d %s accelerators, which is not a multiple of %d per node", run.RunId, count, accelType, perNode)
	}
	return count / perNode, nil
}

// downloadLogs writes each node's logs to <downloadTo>/logs/node_<n>.log and
// prints a summary. An explicit --node downloads only that node; otherwise all of
// them. Logs come from MLflow artifacts, since Bricklens only streams. The
// returned bool is the run's outcome, so the exit code matches the streaming path.
func downloadLogs(ctx context.Context, w *databricks.WorkspaceClient, out io.Writer, req logRequest, status logRunStatus) (bool, error) {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: req.runID})
	if err != nil {
		return false, err
	}

	numNodes, err := resolveNodeCount(run)
	if err != nil {
		return false, err
	}

	nodes := make([]int, 0, numNodes)
	if req.nodeSet {
		if req.node >= numNodes {
			return false, fmt.Errorf("%w %d: run has %d node(s), indexed 0 to %d", errNodeOutOfRange, req.node, numNodes, numNodes-1)
		}
		nodes = append(nodes, req.node)
	} else {
		for n := range numNodes {
			nodes = append(nodes, n)
		}
	}

	// A run with no logs is reported the same way as on the streaming path, so
	// the message and exit code agree between them.
	ids := mlflowIDs(ctx, w, run)
	if ids == nil || ids.RunID == "" {
		emitNoLogs(out, req, status)
		return status.downloadOutcome(), nil
	}

	dir, err := filepath.Abs(req.downloadTo)
	if err != nil {
		return false, err
	}
	// Created up front so a bad --download-to fails with a clear message before
	// any download work happens.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, fmt.Errorf("failed to create %s: %w", dir, err)
	}

	nodeLogs, failures, err := downloadAllNodeLogs(ctx, w, ids.RunID, dir, nodes, req.attempt)
	if err != nil {
		return false, err
	}
	for _, node := range sortedNodeKeys(failures) {
		cmdio.LogString(ctx, fmt.Sprintf("warning: node %d: %s", node, failures[node]))
	}

	if len(nodeLogs) == 0 {
		// "No logs available" would be a lie when the logs exist but couldn't be
		// fetched. The warnings above go to stderr, which a -o json consumer reading
		// stdout never sees, so fail instead of reporting an empty run.
		if len(failures) > 0 {
			return false, fmt.Errorf("failed to download logs from any of %d node(s): %s",
				len(nodes), failures[sortedNodeKeys(failures)[0]])
		}
		emitNoLogs(out, req, status)
		return status.downloadOutcome(), nil
	}

	cmdio.LogString(ctx, fmt.Sprintf("Downloaded logs from %d of %d node(s) to %s", len(nodeLogs), len(nodes), dir))
	for _, node := range sortedNodeKeys(nodeLogs) {
		// Flag it on the file's own line, not just in the warning above.
		suffix := ""
		if _, truncated := failures[node]; truncated {
			suffix = " (incomplete)"
		}
		cmdio.LogString(ctx, fmt.Sprintf("  node %d: %s%s", node, nodeLogs[node], suffix))
	}
	return status.downloadOutcome(), nil
}

// downloadAllNodeLogs downloads the nodes' logs in parallel. It returns a
// node->path map for the nodes that had logs and a node->reason map for those
// that failed; a truncated node appears in both. The log-dir layout is run-wide,
// so it is probed once here rather than by every worker.
func downloadAllNodeLogs(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID, dir string, nodes []int, attempt int) (map[int]string, map[int]string, error) {
	// -1 (latest) maps to attempt 0's directory, as on the streaming path.
	attemptDir := max(attempt, 0)
	withAttempt, err := discoverAttemptPrefix(ctx, w, mlflowRunID, attemptDir)
	if err != nil {
		return nil, nil, err
	}

	paths := make([]string, len(nodes))
	reasons := make([]string, len(nodes))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(downloadConcurrency)
	for i, node := range nodes {
		g.Go(func() error {
			path, err := downloadNodeLog(gctx, w, mlflowRunID, node, attemptDir, withAttempt, dir)
			switch {
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				// Interrupting the command must not look like a node with no logs.
				return err
			case err != nil:
				// One bad node shouldn't abort the rest. A truncated node
				// returns a path as well as an error, so keep both.
				reasons[i] = err.Error()
				paths[i] = path
			default:
				paths[i] = path
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	nodeLogs := map[int]string{}
	failures := map[int]string{}
	for i, node := range nodes {
		if paths[i] != "" {
			nodeLogs[node] = paths[i]
		}
		if reasons[i] != "" {
			failures[node] = reasons[i]
		}
	}
	return nodeLogs, failures, nil
}

// downloadNodeLog streams a node's chunks in order into dir/logs/node_<n>.log,
// returning the path, or "" if the node logged nothing. A failed chunk is skipped
// rather than ending the walk, so a partial download returns both a path and an
// error naming the gaps. The bytes are copied verbatim: a download should
// reproduce the log exactly, so it must not round-trip through lines (which would
// rewrite line endings and cap long lines).
func downloadNodeLog(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string, node, attempt int, withAttempt bool, dir string) (string, error) {
	logDir := constructLogPath(node, attempt, withAttempt)
	chunks, err := listLogChunks(ctx, w, mlflowRunID, logDir)
	if err != nil {
		return "", err
	}
	if len(chunks) == 0 {
		// The listing can lag behind the sidecar, so fall back to chunk 0 as the
		// streaming path does.
		chunks = []logChunk{{index: 0, path: path.Join(logDir, chunkFileName(0))}}
	}

	outPath := filepath.Join(dir, "logs", fmt.Sprintf("node_%d.log", node))
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Skip a failed chunk and keep going: the tail usually holds the failure
	// signature, so losing it to an early bad chunk is worse than a gap. Cancellation
	// still aborts, since every remaining chunk would fail too.
	var written int64
	var missing []int
	for _, chunk := range chunks {
		n, err := copyArtifactTo(ctx, w, mlflowRunID, chunk.path, f)
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			os.Remove(outPath)
			return "", err
		case err != nil:
			log.Debugf(ctx, "air logs: node %d chunk %d failed: %v", node, chunk.index, err)
			missing = append(missing, chunk.index)
		default:
			written += n
		}
	}
	if written == 0 {
		os.Remove(outPath)
		if len(missing) > 0 {
			return "", fmt.Errorf("every chunk failed to download (%d total)", len(missing))
		}
		return "", nil
	}
	if len(missing) > 0 {
		return outPath, fmt.Errorf("incomplete: chunk(s) %v failed to download", missing)
	}
	return outPath, nil
}

// copyArtifactTo streams one artifact's bytes into dst and returns how many were
// written.
func copyArtifactTo(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID, artifactPath string, dst io.Writer) (int64, error) {
	local, err := downloadArtifact(ctx, w, mlflowRunID, artifactPath)
	if err != nil {
		return 0, err
	}
	defer os.Remove(local)

	src, err := os.Open(local)
	if err != nil {
		return 0, err
	}
	defer src.Close()
	return io.Copy(dst, src)
}

// sortedNodeKeys returns the map's node ids in ascending order, so the summary
// prints deterministically.
func sortedNodeKeys(m map[int]string) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
