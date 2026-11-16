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
	"sync"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"golang.org/x/sync/errgroup"
)

// downloadConcurrency caps how many nodes download at once, to avoid hammering
// the artifact store on a wide run.
const downloadConcurrency = 8

func validateNodeIndex(node, numNodes int) error {
	if node < numNodes {
		return nil
	}
	return fmt.Errorf("invalid --node %d: run has %d node(s), indexed 0 to %d", node, numNodes, numNodes-1)
}

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
// them. Logs come from Bricklens unless legacy artifact downloads are enabled. The
// returned bool is the run's outcome, so the exit code matches the streaming path.
func downloadLogs(ctx context.Context, w *databricks.WorkspaceClient, out io.Writer, req logRequest, status logRunStatus, run *jobs.Run) (bool, error) {
	numNodes, err := resolveNodeCount(run)
	if err != nil {
		return false, err
	}

	nodes := make([]int, 0, numNodes)
	if req.nodeSet {
		nodes = append(nodes, req.node)
	} else {
		for n := range numNodes {
			nodes = append(nodes, n)
		}
	}

	// A run with no logs is reported the same way as on the streaming path, so
	// the message and exit code agree between them.
	attempt, taskRunID, err := resolveLogAttempt(run, req.attempt)
	if err != nil {
		return false, err
	}
	var mlflowRunID string
	if mlflowLogFallbackEnabled(ctx) {
		ids := mlflowIDsForTask(ctx, w, taskRunID)
		if ids == nil || ids.RunID == "" {
			emitNoLogs(out, req, status)
			return status.downloadOutcome(), nil
		}
		mlflowRunID = ids.RunID
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

	var nodeLogs map[int]string
	var failures map[int]string
	if mlflowLogFallbackEnabled(ctx) {
		nodeLogs, failures, err = downloadAllNodeLogs(ctx, w, mlflowRunID, dir, nodes, attempt)
	} else {
		req.attempt = attempt
		nodeLogs, failures, err = downloadNodes(ctx, nodes, func(ctx context.Context, node int) (string, error) {
			return downloadBricklensNodeLog(ctx, w, dir, node, req, status)
		})
	}
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

	return downloadNodes(ctx, nodes, func(ctx context.Context, node int) (string, error) {
		return downloadNodeLog(ctx, w, mlflowRunID, node, attemptDir, withAttempt, dir)
	})
}

// downloadNodes bounds concurrency and retains partial files alongside their errors.
func downloadNodes(ctx context.Context, nodes []int, download func(context.Context, int) (string, error)) (map[int]string, map[int]string, error) {
	paths := make([]string, len(nodes))
	reasons := make([]string, len(nodes))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(downloadConcurrency)
	for i, node := range nodes {
		g.Go(func() error {
			path, err := download(gctx, node)
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
	// Idempotent so the deferred cleanup and the explicit close before os.Remove
	// (Windows can't remove an open file) don't double-close.
	closeFile := sync.OnceFunc(func() { _ = f.Close() })
	defer closeFile()

	// Skip a failed chunk and keep going: the tail usually holds the failure
	// signature, so losing it to an early bad chunk is worse than a gap. Cancellation
	// still aborts, since every remaining chunk would fail too.
	var written int64
	var missing []int
	for _, chunk := range chunks {
		n, err := copyArtifactTo(ctx, w, mlflowRunID, chunk.path, f)
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			closeFile()
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
		closeFile()
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

// downloadBricklensNodeLog pages oldest-first without a tail cap. Keep a partial
// file on fetch failure so the caller can report it as incomplete.
func downloadBricklensNodeLog(ctx context.Context, w *databricks.WorkspaceClient, dir string, node int, req logRequest, status logRunStatus) (string, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return "", fmt.Errorf("failed to create API client: %w", err)
	}
	outPath := filepath.Join(dir, "logs", fmt.Sprintf("node_%d.log", node))
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	req.node = node
	st := bricklensStreamer{
		ctx:       ctx,
		w:         w,
		apiClient: apiClient,
		headers:   auth.WorkspaceIDHeaders(w.Config),
		req:       req,
		fromSec:   req.fromSeconds(status, time.Now()),
	}
	var written bool
	var pageToken string
	for {
		var resp *bricklensLogsResponse
		resp, err = st.requestPage(pageToken, req.toSeconds(status), 0, true)
		if err != nil {
			break
		}
		for _, rec := range resp.LogRecords {
			if _, err = fmt.Fprintln(f, rec.Body); err != nil {
				break
			}
			written = true
		}
		if err != nil || resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if ctx.Err() != nil {
		os.Remove(outPath)
		return "", ctx.Err()
	}
	if !written {
		os.Remove(outPath)
		return "", err
	}
	return outPath, err
}
