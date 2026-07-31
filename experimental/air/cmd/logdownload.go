package aircmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"golang.org/x/sync/errgroup"
)

// downloadConcurrency bounds the parallel per-node log downloads.
const downloadConcurrency = 8

// resolveNodeCount returns the number of nodes a run used, derived from its
// accelerator type and count (accelerators are allocated in whole nodes). It
// errors when the run carries no AI runtime compute config.
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

// downloadAllNodeLogs downloads the given nodes' logs in parallel and returns a
// node->file-path map for the nodes that produced logs. The attempt-prefix layout
// is probed once up front so per-node workers skip re-discovery.
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
				// A single node's failure is not fatal to the whole download; log and skip.
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
