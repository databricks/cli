package aircmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

var mlflowArtifactRoots sync.Map

func mlflowArtifactRoot(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID string) (string, error) {
	key := strings.TrimRight(w.Config.Host, "/") + "\x00" + mlflowRunID
	if cached, ok := mlflowArtifactRoots.Load(key); ok {
		return cached.(string), nil
	}
	response, err := w.Experiments.GetRun(ctx, ml.GetRunRequest{RunId: mlflowRunID})
	if err != nil {
		log.Debugf(ctx, "air logs: could not resolve MLflow artifact URI for %s: %v", mlflowRunID, err)
		return "", nil
	}
	if response.Run == nil || response.Run.Info == nil {
		return "", nil
	}
	root := strings.TrimRight(response.Run.Info.ArtifactUri, "/")
	mlflowArtifactRoots.Store(key, root)
	return root, nil
}

func volumeArtifactRoot(root string) (string, bool) {
	if strings.HasPrefix(root, "dbfs:/Volumes/") {
		return strings.TrimPrefix(root, "dbfs:"), true
	}
	return "", false
}

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

	if !status.terminal() && !req.staticView {
		return streamMLflowLogs(ctx, w, out, req, status)
	}
	return fetchMLflowLogTail(ctx, w, out, req, status)
}

func fetchMLflowLogTail(ctx context.Context, w *databricks.WorkspaceClient, out io.Writer, req logRequest, status logRunStatus) (bool, error) {
	mlflowRunID, logDir, err := resolveMLflowLogPath(ctx, w, req)
	if err != nil {
		return false, err
	}
	if mlflowRunID == "" || logDir == "" {
		emitNoLogs(out, req, status)
		return status.downloadOutcome(), nil
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
		return status.downloadOutcome(), nil
	}

	lines, err := tailChunks(ctx, w, mlflowRunID, chunks, target)
	if err != nil {
		return false, err
	}
	if len(lines) == 0 {
		emitNoLogs(out, req, status)
		return status.downloadOutcome(), nil
	}

	if len(lines) > target {
		lines = lines[len(lines)-target:]
	}
	for _, line := range lines {
		emitLogLine(out, req, line)
	}
	return status.downloadOutcome(), nil
}

func mlflowLogsExist(ctx context.Context, w *databricks.WorkspaceClient, req logRequest) bool {
	mlflowRunID, logDir, err := resolveMLflowLogPath(ctx, w, req)
	if err != nil || mlflowRunID == "" || logDir == "" {
		if err != nil {
			log.Debugf(ctx, "air logs: MLflow log-existence probe failed for run %d: %v", req.runID, err)
		}
		return false
	}
	chunks, err := listLogChunks(ctx, w, mlflowRunID, logDir)
	if err != nil {
		log.Debugf(ctx, "air logs: MLflow log-existence probe failed for run %d: %v", req.runID, err)
		return false
	}
	return len(chunks) > 0
}

func streamMLflowLogs(ctx context.Context, w *databricks.WorkspaceClient, out io.Writer, req logRequest, status logRunStatus) (bool, error) {
	lineOffsets := make(map[string]int)
	currentRunID := ""
	printed := false
	previousState := ""

	for {
		if req.onStatusChange != nil {
			current := status.displayState()
			if current != previousState {
				req.onStatusChange(current, previousState)
				previousState = current
			}
		}

		mlflowRunID, logDir, err := resolveMLflowLogPath(ctx, w, req)
		if err != nil {
			if status.terminal() {
				return false, err
			}
			log.Debugf(ctx, "air logs: failed to resolve MLflow log path: %v", err)
		} else if mlflowRunID != "" && logDir != "" {
			if currentRunID != mlflowRunID {
				currentRunID = mlflowRunID
				lineOffsets = make(map[string]int)
			}
			chunks, listErr := listLogChunks(ctx, w, mlflowRunID, logDir)
			if listErr != nil {
				if status.terminal() {
					return false, listErr
				}
				log.Debugf(ctx, "air logs: failed to list MLflow log chunks: %v", listErr)
			} else {
				for _, chunk := range chunks {
					lines, downloadErr := downloadChunkLines(ctx, w, mlflowRunID, chunk.path)
					if downloadErr != nil {
						if status.terminal() {
							return false, downloadErr
						}
						log.Debugf(ctx, "air logs: failed to read MLflow log chunk %s: %v", chunk.path, downloadErr)
						continue
					}
					offset := lineOffsets[chunk.path]
					if offset > len(lines) {
						offset = 0
					}
					for _, line := range lines[offset:] {
						emitLogLine(out, req, line)
						printed = true
					}
					lineOffsets[chunk.path] = len(lines)
				}
			}
		}

		if status.terminal() {
			if !printed {
				emitNoLogs(out, req, status)
			}
			return status.succeeded(), nil
		}
		if err := sleepOrCancel(ctx, retryCheckInterval); err != nil {
			return false, err
		}
		refreshed, err := resolveRunStatus(ctx, w, req.runID)
		if err != nil {
			if errors.Is(err, apierr.ErrResourceDoesNotExist) || ctx.Err() != nil {
				return false, err
			}
			log.Debugf(ctx, "air logs: failed to refresh run status on MLflow fallback: %v", err)
			continue
		}
		status = refreshed
	}
}

// resolveMLflowLogPath returns the run's MLflow run id and per-node log directory.
func resolveMLflowLogPath(ctx context.Context, w *databricks.WorkspaceClient, req logRequest) (string, string, error) {
	run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: req.runID})
	if err != nil {
		return "", "", err
	}
	attempt, taskRunID, err := resolveLogAttempt(run, req.attempt)
	if err != nil {
		return "", "", err
	}
	ids := mlflowIDsForTask(ctx, w, taskRunID)
	if ids == nil || ids.RunID == "" {
		return "", "", nil
	}

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
func listArtifacts(ctx context.Context, w *databricks.WorkspaceClient, mlflowRunID, artifactPath string) ([]ml.FileInfo, error) {
	root, err := mlflowArtifactRoot(ctx, w, mlflowRunID)
	if err != nil {
		return nil, err
	}
	if volumeRoot, ok := volumeArtifactRoot(root); ok {
		fc, err := filer.NewWorkspaceFilesClient(w, volumeRoot)
		if err != nil {
			return nil, err
		}
		entries, err := fc.ReadDir(ctx, artifactPath)
		if err != nil {
			return nil, err
		}
		files := make([]ml.FileInfo, 0, len(entries))
		for _, entry := range entries {
			info := ml.FileInfo{Path: path.Join(artifactPath, entry.Name()), IsDir: entry.IsDir()}
			if !entry.IsDir() {
				stat, err := entry.Info()
				if err != nil {
					return nil, err
				}
				info.FileSize = stat.Size()
			}
			files = append(files, info)
		}
		return files, nil
	}
	it := w.Experiments.ListArtifacts(ctx, ml.ListArtifactsRequest{RunId: mlflowRunID, Path: artifactPath})
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
	root, err := mlflowArtifactRoot(ctx, w, mlflowRunID)
	if err != nil {
		return "", err
	}
	if volumeRoot, ok := volumeArtifactRoot(root); ok {
		fc, err := filer.NewWorkspaceFilesClient(w, volumeRoot)
		if err != nil {
			return "", err
		}
		reader, err := fc.Read(ctx, artifactPath)
		if err != nil {
			return "", err
		}
		defer reader.Close()
		tmp, err := os.CreateTemp("", "air-log-chunk-*")
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(tmp, reader); err != nil {
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
