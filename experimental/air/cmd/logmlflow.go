package aircmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// chunkFilePattern matches a sidecar log chunk file (logs-<index>.chunk.txt);
// group 1 is the chunk index. The sidecar splits the consolidated stdout stream
// into 4MB chunks, advancing the index monotonically.
var chunkFilePattern = regexp.MustCompile(`^logs-(\d+)\.chunk\.txt$`)

// oldFormatNodeDir matches a bare per-node log dir (logs/node_<n>) at the top of
// the run's logs/ dir. The attempt-prefixed layout nests these under
// logs/attempt_<n>/, so a bare logs/node_<n> only appears in the old layout.
var oldFormatNodeDir = regexp.MustCompile(`^logs/node_\d+$`)

// mlflowLogFallback prints a run's logs from MLflow artifacts. It is the fallback
// used when Bricklens can't serve logs (errBricklensFeatureDisabled).
//
// It resolves the run's MLflow run id, discovers the per-node log directory
// (logs/node_Y or logs/attempt_X/node_Y), lists the chunk files, and walks them
// newest-first until it has the requested tail, then prints oldest-first. This
// mirrors the Python print_all_logs path.
//
// The tail length is --lines when set, else the default cap. MLflow chunk files
// are not time-indexed, so --minutes cannot restrict the window here (Bricklens
// is the time-indexed backend); when only --minutes is set the default tail cap
// applies, and a debug line notes the flag was inapplicable on this path.
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
		return false, nil
	}

	chunks, err := listLogChunks(ctx, w, mlflowRunID, logDir)
	if err != nil {
		return false, err
	}
	if len(chunks) == 0 {
		// Nothing listed yet: assume the single chunk 0, matching the Python
		// legacy single-chunk fallback.
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
		return false, nil
	}

	if len(lines) > target {
		lines = lines[len(lines)-target:]
	}
	for _, line := range lines {
		emitLogLine(out, req, line)
	}
	return status.succeeded(), nil
}

// resolveMLflowLogPath returns the run's MLflow run id and per-node log
// directory. The directory format (attempt-prefixed or not) is deployment-wide,
// so it is discovered from the artifact listing under logs/.
func resolveMLflowLogPath(ctx context.Context, w *databricks.WorkspaceClient, req logRequest) (string, string, error) {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: req.runID})
	if err != nil {
		return "", "", err
	}
	ids := mlflowIDs(ctx, w, run)
	if ids == nil || ids.RunID == "" {
		return "", "", nil
	}

	// attempt is -1 (latest) or an explicit retry; the log path uses attempt 0
	// when latest, matching how the sidecar names the first attempt's dir.
	attempt := max(req.attempt, 0)
	withAttempt, err := discoverAttemptPrefix(ctx, w, ids.RunID, attempt)
	if err != nil {
		return "", "", err
	}
	return ids.RunID, constructLogPath(req.node, attempt, withAttempt), nil
}

// discoverAttemptPrefix probes the run's logs/ dir once to decide whether the
// deployment uses the attempt-prefixed layout (logs/attempt_X/node_Y) or the old
// one (logs/node_Y). A bare logs/node_<n> anywhere means old format; otherwise a
// logs/attempt_<attempt> entry means the new format. Defaults to old format when
// nothing is listed yet.
func discoverAttemptPrefix(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string, attempt int) (bool, error) {
	files, err := listArtifacts(ctx, w, mlflowRunID, "logs")
	if err != nil {
		// A listing failure here is not fatal: fall back to the old (no-attempt)
		// layout, which the chunk listing will simply find empty if wrong.
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

// listLogChunks lists the chunk files under a per-node log dir, sorted ascending
// by index. Non-chunk entries are ignored.
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

// tailChunks walks the chunks newest-first, prepending each chunk's lines, until
// it has at least `target` lines (or runs out). This is bandwidth-optimal for the
// common case where the tail fits in the last chunk or two. If a chunk fails to
// download mid-walk, it stops rather than splice non-adjacent chunks.
func tailChunks(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string, chunks []logChunk, target int) ([]string, error) {
	var accumulated []string
	for _, chunk := range slices.Backward(chunks) {
		lines, err := downloadChunkLines(ctx, w, mlflowRunID, chunk.path)
		if err != nil {
			log.Debugf(ctx, "air logs: failed to download chunk %d; showing only logs after the gap: %v", chunk.index, err)
			break
		}
		accumulated = append(lines, accumulated...)
		if len(accumulated) >= target {
			break
		}
	}
	return accumulated, nil
}

// downloadChunkLines fetches one chunk artifact and returns its lines. The bytes
// are read via the credential-vended download so large log files are not capped
// by the proxied get-artifact endpoint.
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

// listArtifacts lists a run's artifacts under a path via the typed SDK iterator.
func listArtifacts(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID, path string) ([]ml.FileInfo, error) {
	it := w.Experiments.ListArtifacts(ctx, ml.ListArtifactsRequest{RunId: mlflowRunID, Path: path})
	return listing.ToSlice(ctx, it)
}

// credentialInfo is one entry of the credentials-for-read response: a pre-signed
// URL for a run artifact plus any backend-required request headers.
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

// downloadArtifact downloads a single run artifact to a temp file and returns its
// path. It uses the credential-vending flow the MLflow SDK uses for
// Databricks-backed runs: credentials-for-read returns a pre-signed URL, which we
// stream to disk. The credentials-for-read endpoint is not modeled by the SDK, so
// it is called via a raw client.Do.
func downloadArtifact(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID, artifactPath string) (string, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return "", fmt.Errorf("failed to create API client: %w", err)
	}

	var resp credentialsForReadResponse
	query := map[string]any{
		"run_id": mlflowRunID,
		// path is a repeated field; the SDK serializes a slice as path=...&path=...
		"path": []string{artifactPath},
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
	// Azure SAS / some GCS URIs require backend-supplied headers; AWS pre-signed
	// URLs return none.
	for _, h := range cred.Headers {
		req.Header.Set(h.Name, h.Value)
	}

	httpResp, err := http.DefaultClient.Do(req)
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
