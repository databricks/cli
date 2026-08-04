package aircmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// chunkFilePattern matches a log chunk file (logs-<index>.chunk.txt); group 1 is
// the chunk index. The sidecar splits stdout into 4MB chunks, index ascending.
var chunkFilePattern = regexp.MustCompile(`^logs-(\d+)\.chunk\.txt$`)

// oldFormatNodeDir matches a bare per-node log dir (logs/node_<n>). The
// attempt-prefixed layout nests these under logs/attempt_<n>/, so a bare
// logs/node_<n> only appears in the old layout.
var oldFormatNodeDir = regexp.MustCompile(`^logs/node_\d+$`)

// artifactDownloadClient fetches pre-signed artifact URLs with connect and
// response-header timeouts, so a stalled storage backend can't hang the command.
// Mirrors the Python CLI's (10s connect, 60s read) bounds.
var artifactDownloadClient = &http.Client{
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		ResponseHeaderTimeout: 60 * time.Second,
	},
}

// mlflowLogFallback prints a run's logs from MLflow artifacts, the fallback when
// Bricklens can't serve them. It resolves the MLflow run id, discovers the
// per-node log directory, lists the chunk files, and walks them newest-first
// until it has the requested tail, then prints oldest-first.
//
// The tail length is --lines, else the default cap. MLflow chunks are not
// time-indexed, so --minutes cannot restrict the window here.
func mlflowLogFallback(ctx context.Context, w *databricks.WorkspaceClient, out io.Writer, req logRequest, status logRunStatus) (bool, error) {
	if req.windowMinutes > 0 {
		log.Debugf(ctx, "air logs: --minutes is not supported on the MLflow fallback path; showing the default tail")
	}

	mlflowRunID, logDir, err := resolveMLflowLogPath(ctx, w, req)
	if err != nil {
		return false, err
	}
	if mlflowRunID == "" || logDir == "" {
		emitNoLogs(out, req, status)
		return status.succeeded(), nil
	}

	chunks, err := listLogChunks(ctx, w, mlflowRunID, logDir)
	if err != nil {
		return false, err
	}
	if len(chunks) == 0 {
		// Nothing listed yet: assume the single chunk 0.
		chunks = []logChunk{{index: 0, path: path.Join(logDir, chunkFileName(0))}}
	}

	target := req.tailTarget()
	if target <= 0 {
		return status.succeeded(), nil
	}

	lines, err := tailChunks(ctx, w, mlflowRunID, chunks, target)
	if err != nil {
		return false, err
	}
	if len(lines) == 0 {
		emitNoLogs(out, req, status)
		return status.succeeded(), nil
	}

	if len(lines) > target {
		lines = lines[len(lines)-target:]
	}
	for _, line := range lines {
		emitLogLine(out, req, line)
	}
	return status.succeeded(), nil
}

// resolveMLflowLogPath returns the run's MLflow run id and per-node log directory.
func resolveMLflowLogPath(ctx context.Context, w *databricks.WorkspaceClient, req logRequest) (string, string, error) {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: req.runID})
	if err != nil {
		return "", "", err
	}
	ids := mlflowIDs(ctx, w, run)
	if ids == nil || ids.RunID == "" {
		return "", "", nil
	}

	// -1 (latest) maps to attempt 0's directory.
	attempt := max(req.attempt, 0)
	withAttempt, err := discoverAttemptPrefix(ctx, w, ids.RunID, attempt)
	if err != nil {
		return "", "", err
	}
	return ids.RunID, constructLogPath(req.node, attempt, withAttempt), nil
}

// discoverAttemptPrefix probes the logs/ dir once to decide whether the layout is
// attempt-prefixed (logs/attempt_X/node_Y) or old (logs/node_Y). A bare
// logs/node_<n> means old; a logs/attempt_<attempt> entry means prefixed.
// Defaults to old when nothing is listed.
func discoverAttemptPrefix(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string, attempt int) (bool, error) {
	files, err := listArtifacts(ctx, w, mlflowRunID, "logs")
	if err != nil {
		// Not fatal: default to the old layout; the chunk listing finds it empty if wrong.
		log.Debugf(ctx, "air logs: could not list logs dir for format discovery: %v", err)
		return false, nil
	}

	attemptDir := fmt.Sprintf("logs/attempt_%d", attempt)
	for _, f := range files {
		if oldFormatNodeDir.MatchString(f.Path) {
			return false, nil
		}
		if f.Path == attemptDir {
			return true, nil
		}
	}
	return false, nil
}

// constructLogPath builds the per-node log directory for a node and attempt.
func constructLogPath(node, attempt int, withAttempt bool) string {
	if withAttempt {
		return fmt.Sprintf("logs/attempt_%d/node_%d", attempt, node)
	}
	return fmt.Sprintf("logs/node_%d", node)
}

// chunkFileName is the artifact filename for a chunk index.
func chunkFileName(index int) string {
	return fmt.Sprintf("logs-%d.chunk.txt", index)
}

// logChunk is one listed chunk: its index and full artifact path.
type logChunk struct {
	index int
	path  string
}

// listLogChunks lists the chunk files under a log dir, sorted ascending by index.
func listLogChunks(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID, logDir string) ([]logChunk, error) {
	files, err := listArtifacts(ctx, w, mlflowRunID, logDir)
	if err != nil {
		return nil, err
	}

	var chunks []logChunk
	for _, f := range files {
		base := path.Base(f.Path)
		m := chunkFilePattern.FindStringSubmatch(base)
		if m == nil {
			continue
		}
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		chunks = append(chunks, logChunk{index: idx, path: f.Path})
	}
	slices.SortFunc(chunks, func(a, b logChunk) int { return a.index - b.index })
	return chunks, nil
}

// tailChunks walks chunks newest-first, prepending each chunk's lines, until it
// has `target` lines or runs out. A mid-walk download failure stops the walk
// rather than splice non-adjacent chunks.
func tailChunks(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string, chunks []logChunk, target int) ([]string, error) {
	var accumulated []string
	for _, chunk := range slices.Backward(chunks) {
		lines, err := downloadChunkLines(ctx, w, mlflowRunID, chunk.path)
		if err != nil {
			log.Debugf(ctx, "air logs: failed to download chunk %d; showing only logs after it: %v", chunk.index, err)
			break
		}
		accumulated = append(lines, accumulated...)
		if len(accumulated) >= target {
			break
		}
	}
	return accumulated, nil
}

// downloadChunkLines fetches one chunk artifact and returns its lines.
func downloadChunkLines(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID, artifactPath string) ([]string, error) {
	f, err := downloadArtifact(ctx, w, mlflowRunID, artifactPath)
	if err != nil {
		return nil, err
	}
	defer os.Remove(f)

	file, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// listArtifacts lists a run's artifacts under a path.
func listArtifacts(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID, path string) ([]ml.FileInfo, error) {
	it := w.Experiments.ListArtifacts(ctx, ml.ListArtifactsRequest{RunId: mlflowRunID, Path: path})
	return listing.ToSlice(ctx, it)
}

// credentialInfo is one credentials-for-read entry: a pre-signed URL plus any
// backend-required request headers.
type credentialInfo struct {
	SignedURI string `json:"signed_uri"`
	Headers   []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"headers"`
}

type credentialsForReadResponse struct {
	CredentialInfos []credentialInfo `json:"credential_infos"`
}

// downloadArtifact downloads one run artifact to a temp file and returns its
// path. credentials-for-read returns a pre-signed URL, which we stream to disk;
// that endpoint is not modeled by the SDK, so it is called via a raw client.Do.
func downloadArtifact(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID, artifactPath string) (string, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return "", fmt.Errorf("failed to create API client: %w", err)
	}

	var resp credentialsForReadResponse
	// A map query is serialized per value with %v, so a []string becomes the
	// literal "[path]". The backend signs that bogus path and still returns 200,
	// surfacing only as a 404 on the download.
	query := map[string]any{
		"run_id": mlflowRunID,
		"path":   artifactPath,
	}
	err = apiClient.Do(ctx, http.MethodGet, "/api/2.0/mlflow/artifacts/credentials-for-read", nil, nil, query, &resp)
	if err != nil {
		return "", fmt.Errorf("failed to get read credentials for %s: %w", artifactPath, err)
	}
	if len(resp.CredentialInfos) == 0 || resp.CredentialInfos[0].SignedURI == "" {
		return "", fmt.Errorf("no download credentials returned for %s", artifactPath)
	}
	cred := resp.CredentialInfos[0]

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cred.SignedURI, nil)
	if err != nil {
		return "", err
	}
	// Azure SAS / some GCS URIs require backend-supplied headers; AWS returns none.
	for _, h := range cred.Headers {
		req.Header.Set(h.Name, h.Value)
	}

	httpResp, err := artifactDownloadClient.Do(req)
	if err != nil {
		return "", err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode >= 400 {
		return "", fmt.Errorf("artifact download failed: HTTP %d", httpResp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "air-log-chunk-*")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(tmp, httpResp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}
