package aircmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"golang.org/x/sync/errgroup"
)

// downloadConcurrency bounds the parallel per-node log downloads.
const downloadConcurrency = 8

// resolveNodeCount returns the number of nodes a run used, derived from its
// accelerator type and count (accelerators are allocated in whole nodes). It
// errors when the run carries no AIR compute.
func resolveNodeCount(run *jobs.Run) (int, error) {
	accelType, count := jobCompute(run)
	if accelType == "" || count <= 0 {
		return 0, fmt.Errorf("run %d has no AI runtime compute config", run.RunId)
	}
	g, err := parseGPUType(accelType)
	if err != nil {
		return 0, err
	}
	perNode, err := gpusPerNode(g)
	if err != nil {
		return 0, err
	}
	return count / perNode, nil
}

// downloadLogs writes each node's full logs to disk under dir/logs/node_<n>.log
// and prints a summary. With an explicit --node it downloads just that node,
// else every node. Logs come from the MLflow artifact store (the download path
// has no Bricklens equivalent). Returns whether the run succeeded, so the exit
// code matches the streaming path.
func downloadLogs(ctx context.Context, w *databricks.WorkspaceClient, req logRequest, status logRunStatus, dir string) (bool, error) {
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
			return false, fmt.Errorf("node %d does not exist: run has %d node(s), indexed 0 to %d", req.node, numNodes, numNodes-1)
		}
		nodes = append(nodes, req.node)
	} else {
		for n := range numNodes {
			nodes = append(nodes, n)
		}
	}

	ids := mlflowIDs(ctx, w, run)
	if ids == nil || ids.RunID == "" {
		return status.succeeded(), fmt.Errorf("no logs available for run %d", req.runID)
	}

	dir, err = filepath.Abs(dir)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, fmt.Errorf("failed to create %s: %w", dir, err)
	}

	nodeLogs, err := downloadAllNodeLogs(ctx, w, ids.RunID, dir, nodes, req.attempt)
	if err != nil {
		return false, err
	}
	if len(nodeLogs) == 0 {
		return status.succeeded(), fmt.Errorf("no logs available for run %d", req.runID)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Downloaded logs from %d of %d node(s) to %s", len(nodeLogs), len(nodes), filepath.ToSlash(dir)))
	for _, node := range sortedKeys(nodeLogs) {
		cmdio.LogString(ctx, fmt.Sprintf("  node %d: %s", node, filepath.ToSlash(nodeLogs[node])))
	}
	return status.succeeded(), nil
}

// downloadAllNodeLogs downloads the given nodes' logs in parallel and returns a
// node->file-path map for the nodes that produced logs. withAttempt is probed
// once up front so per-node workers skip re-discovery.
func downloadAllNodeLogs(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID, dir string, nodes []int, attempt int) (map[int]string, error) {
	// -1 (latest) maps to attempt 0's directory, matching the streaming path.
	attemptDir := max(attempt, 0)
	withAttempt, err := discoverAttemptPrefix(ctx, w, mlflowRunID, attemptDir)
	if err != nil {
		return nil, err
	}

	paths := make([]string, len(nodes))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(downloadConcurrency)
	for i, node := range nodes {
		g.Go(func() error {
			path, err := downloadNodeLog(gctx, w, mlflowRunID, node, attemptDir, withAttempt, dir)
			if err != nil {
				// A node with no logs is not fatal; log and skip it.
				log.Debugf(gctx, "air logs: node %d download failed: %v", node, err)
				return nil
			}
			paths[i] = path
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	nodeLogs := map[int]string{}
	for i, node := range nodes {
		if paths[i] != "" {
			nodeLogs[node] = paths[i]
		}
	}
	return nodeLogs, nil
}

// downloadNodeLog concatenates a node's chunk files (ascending) into
// dir/logs/node_<n>.log and returns the file path, or "" when the node produced
// no logs.
func downloadNodeLog(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string, node, attempt int, withAttempt bool, dir string) (string, error) {
	logDir := constructLogPath(node, attempt, withAttempt)
	chunks, err := listLogChunks(ctx, w, mlflowRunID, logDir)
	if err != nil {
		return "", err
	}
	if len(chunks) == 0 {
		return "", nil
	}

	var b strings.Builder
	for _, chunk := range chunks {
		lines, err := downloadChunkLines(ctx, w, mlflowRunID, chunk.path)
		if err != nil {
			return "", err
		}
		for _, line := range lines {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	outPath := filepath.Join(dir, "logs", fmt.Sprintf("node_%d.log", node))
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return outPath, nil
}

// sortedKeys returns a map's int keys in ascending order, for deterministic output.
func sortedKeys(m map[int]string) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
