package aircmd

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/klauspost/pgzip"
)

const (
	snapshotOverlayVersion        = "v1"
	snapshotOverlayDescriptorName = ".databricks/air-source-overlay-v1/descriptor"
	snapshotOverlayDeletionsRoot  = ".databricks/air-source-overlay-v1/deletions"

	snapshotAnchorMaxAge            = 30 * 24 * time.Hour
	snapshotOverlayMaxChanged       = 1024
	snapshotOverlayMaxBytes   int64 = 128 << 20
	snapshotOverlayMaxRatio         = 0.25
)

type snapshotOverlayEntry struct {
	Size       int64  `json:"size"`
	ModTime    int64  `json:"mtime_ns"`
	Mode       uint32 `json:"mode"`
	LinkTarget string `json:"link_target,omitempty"`
}

type snapshotAnchorState struct {
	Version           string                          `json:"version"`
	Branch            string                          `json:"branch"`
	CreatedAtUnix     int64                           `json:"created_at_unix"`
	AnchorName        string                          `json:"anchor_name"`
	AnchorFingerprint string                          `json:"anchor_fingerprint"`
	Component         string                          `json:"component"`
	Entries           map[string]snapshotOverlayEntry `json:"entries"`
	Results           map[string]string               `json:"results"`
}

type snapshotOverlayDescriptor struct {
	FormatVersion     int    `json:"format_version"`
	Anchor            string `json:"anchor"`
	AnchorFingerprint string `json:"anchor_fingerprint"`
	ResultFingerprint string `json:"result_fingerprint"`
	Component         string `json:"component"`
}

type snapshotOverlayDiff struct {
	Changed      []snapshotFile
	Deleted      []string
	ChangedBytes int64
	TotalBytes   int64
}

func snapshotOverlayManifest(files []snapshotFile) map[string]snapshotOverlayEntry {
	entries := make(map[string]snapshotOverlayEntry, len(files))
	for _, file := range files {
		entries[filepath.ToSlash(file.rel)] = snapshotOverlayEntry{
			Size:       file.size,
			ModTime:    file.modTime,
			Mode:       file.mode,
			LinkTarget: file.linkTarget,
		}
	}
	return entries
}

func snapshotOverlayFingerprint(files []snapshotFile) string {
	return computePlainTarKey(files)
}

func computeSnapshotOverlayDiff(files []snapshotFile, anchor map[string]snapshotOverlayEntry) snapshotOverlayDiff {
	current := make(map[string]struct{}, len(files))
	diff := snapshotOverlayDiff{}
	for _, file := range files {
		rel := filepath.ToSlash(file.rel)
		current[rel] = struct{}{}
		diff.TotalBytes += file.size
		entry, ok := anchor[rel]
		if !ok || entry.Size != file.size || entry.ModTime != file.modTime ||
			entry.Mode != file.mode || entry.LinkTarget != file.linkTarget {
			diff.Changed = append(diff.Changed, file)
			diff.ChangedBytes += file.size
		}
	}
	for rel := range anchor {
		if _, ok := current[rel]; !ok {
			diff.Deleted = append(diff.Deleted, rel)
		}
	}
	slices.SortFunc(diff.Changed, func(a, b snapshotFile) int {
		return strings.Compare(filepath.ToSlash(a.rel), filepath.ToSlash(b.rel))
	})
	slices.Sort(diff.Deleted)
	return diff
}

func snapshotOverlayAnchorReason(diff snapshotOverlayDiff) string {
	changedPaths := len(diff.Changed) + len(diff.Deleted)
	if changedPaths > snapshotOverlayMaxChanged {
		return "changed-path-count"
	}
	if diff.ChangedBytes > snapshotOverlayMaxBytes {
		return "changed-bytes"
	}
	if diff.TotalBytes > 0 && float64(diff.ChangedBytes)/float64(diff.TotalBytes) > snapshotOverlayMaxRatio {
		return "changed-byte-ratio"
	}
	return ""
}

func snapshotOverlayStatePath(repoPath, configPath string, includePaths []string, branch string) (string, error) {
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return "", err
	}
	branchKey := sha256.Sum256([]byte(snapshotOverlayVersion + "\x00" + branch))
	return filepath.Join(
		snapshotCacheDir(absRepo, absConfig, includePaths),
		"overlay-anchor-"+hex.EncodeToString(branchKey[:8])+".json",
	), nil
}

func loadSnapshotAnchorState(statePath, branch, component string) *snapshotAnchorState {
	file, err := os.Open(statePath)
	if err != nil {
		return nil
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64<<20))
	decoder.DisallowUnknownFields()
	var state snapshotAnchorState
	if err := decoder.Decode(&state); err != nil {
		return nil
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF || state.Version != snapshotOverlayVersion ||
		state.Branch != branch || state.Component != component || state.CreatedAtUnix <= 0 ||
		state.CreatedAtUnix > time.Now().Add(time.Hour).Unix() ||
		state.AnchorName == "" || state.AnchorFingerprint == "" || state.Entries == nil || state.Results == nil {
		return nil
	}
	if !validSnapshotObjectNameForKind(state.AnchorName, "anchor") || !validSnapshotFingerprint(state.AnchorFingerprint) {
		return nil
	}
	anchorFiles := make([]snapshotFile, 0, len(state.Entries))
	for rel, entry := range state.Entries {
		if validateSnapshotRelativePath(rel) != nil {
			return nil
		}
		mode := os.FileMode(entry.Mode)
		if entry.Size < 0 || (!mode.IsRegular() && mode&os.ModeSymlink == 0) ||
			(mode&os.ModeSymlink == 0 && entry.LinkTarget != "") {
			return nil
		}
		anchorFiles = append(anchorFiles, snapshotFile{
			rel:        filepath.FromSlash(rel),
			size:       entry.Size,
			modTime:    entry.ModTime,
			mode:       entry.Mode,
			linkTarget: entry.LinkTarget,
		})
	}
	if snapshotOverlayFingerprint(anchorFiles) != state.AnchorFingerprint || state.Results[state.AnchorFingerprint] != state.AnchorName {
		return nil
	}
	for fingerprint, name := range state.Results {
		if !validSnapshotFingerprint(fingerprint) || !validSnapshotObjectName(name) {
			return nil
		}
	}
	return &state
}

func validSnapshotObjectName(name string) bool {
	return validSnapshotObjectNameForKind(name, "anchor") || validSnapshotObjectNameForKind(name, "overlay")
}

func validSnapshotObjectNameForKind(name, kind string) bool {
	if len(name) > 255 || path.Base(name) != name || strings.ContainsAny(name, "/\\") || !strings.HasSuffix(name, ".tar.gz") {
		return false
	}
	prefix := "air_" + kind + "_" + snapshotOverlayVersion + "_"
	stem := strings.TrimSuffix(name, ".tar.gz")
	return strings.HasPrefix(stem, prefix) && validSnapshotFingerprint(strings.TrimPrefix(stem, prefix))
}

func validSnapshotFingerprint(fingerprint string) bool {
	if len(fingerprint) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(fingerprint)
	return err == nil
}

func saveSnapshotAnchorState(statePath string, state snapshotAnchorState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return fmt.Errorf("failed to create overlay state directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(statePath), "overlay-anchor-*.json")
	if err != nil {
		return fmt.Errorf("failed to write overlay state: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("failed to write overlay state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to write overlay state: %w", err)
	}
	if err := os.Rename(tmpName, statePath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to install overlay state: %w", err)
	}
	return nil
}

// maybeUploadSnapshotOverlay handles the WSFS incremental path. A false
// handled result leaves the caller on the existing full-snapshot path.
func maybeUploadSnapshotOverlay(
	ctx context.Context,
	store filer.Filer,
	uploadPath string,
	repoPath string,
	configPath string,
	plan snapshotPlan,
	files []snapshotFile,
	tmp string,
	artifactPath string,
	noCache bool,
) (result snapshotResult, handled bool, err error) {
	isVolumePath := artifactPath == "/Volumes" || strings.HasPrefix(artifactPath, "/Volumes/")
	if noCache || isVolumePath || plan.mode != modePlainTar {
		return snapshotResult{}, false, nil
	}

	var totalBytes int64
	for _, file := range files {
		totalBytes += file.size
	}
	if totalBytes < snapshotCacheMinBytes || filepath.Base(repoPath) == ".databricks" {
		log.Debugf(ctx, "air snapshot overlay: decision=full-existing reason=small-or-reserved files=%d uncompressed_bytes=%d", len(files), totalBytes)
		return snapshotResult{}, false, nil
	}

	branch := ""
	if plan.isGitRepo {
		branch = newGitRepo(repoPath).currentBranch(ctx)
	}
	component := filepath.Base(repoPath)
	statePath, err := snapshotOverlayStatePath(repoPath, configPath, plan.includePaths, branch)
	if err != nil {
		return snapshotResult{}, true, err
	}
	fingerprint := snapshotOverlayFingerprint(files)
	state := loadSnapshotAnchorState(statePath, branch, component)
	anchorReason := ""
	if state == nil {
		anchorReason = "missing-or-invalid-state"
	} else if time.Since(time.Unix(state.CreatedAtUnix, 0)) >= snapshotAnchorMaxAge {
		anchorReason = "anchor-age"
	} else {
		exists, statErr := snapshotExists(ctx, store, state.AnchorName)
		if statErr != nil {
			return snapshotResult{}, true, statErr
		}
		if !exists {
			anchorReason = "missing-remote-anchor"
		}
	}

	if anchorReason == "" {
		if name, ok := state.Results[fingerprint]; ok {
			exists, statErr := snapshotExists(ctx, store, name)
			if statErr != nil {
				return snapshotResult{}, true, statErr
			}
			if exists {
				log.Debugf(ctx, "air snapshot overlay: decision=reuse branch=%q object=%s", branch, name)
				return snapshotResult{CodeSourcePath: path.Join(uploadPath, name)}, true, nil
			}
			delete(state.Results, fingerprint)
		}

		diff := computeSnapshotOverlayDiff(files, state.Entries)
		anchorReason = snapshotOverlayAnchorReason(diff)
		changedPaths := len(diff.Changed) + len(diff.Deleted)
		ratio := 0.0
		if diff.TotalBytes > 0 {
			ratio = float64(diff.ChangedBytes) / float64(diff.TotalBytes)
		}
		log.Debugf(ctx, "air snapshot overlay: decision-input branch=%q changed_paths=%d changed_bytes=%d total_bytes=%d changed_ratio=%.4f reanchor_reason=%q",
			branch, changedPaths, diff.ChangedBytes, diff.TotalBytes, ratio, anchorReason)
		if anchorReason == "" {
			archivePath := filepath.Join(tmp, "overlay.tar.gz")
			packageStart := time.Now()
			if err := createSnapshotOverlay(repoPath, component, state, fingerprint, diff, archivePath); err != nil {
				return snapshotResult{}, true, err
			}
			afterFiles, err := snapshotFiles(ctx, repoPath, plan.includePaths, plan.isGitRepo)
			if err != nil {
				return snapshotResult{}, true, err
			}
			if snapshotOverlayFingerprint(afterFiles) != fingerprint {
				return snapshotResult{}, true, errors.New("code source changed while creating its overlay snapshot; retry submission")
			}
			packageDuration := time.Since(packageStart)
			archiveInfo, err := os.Stat(archivePath)
			if err != nil {
				return snapshotResult{}, true, err
			}
			publishStart := time.Now()
			name, err := publishImmutableSnapshot(ctx, store, "air_overlay_"+snapshotOverlayVersion, archivePath)
			if err != nil {
				return snapshotResult{}, true, err
			}
			log.Debugf(ctx, "air snapshot overlay: decision=overlay package=%s publish=%s compressed_bytes=%d object=%s",
				packageDuration, time.Since(publishStart), archiveInfo.Size(), name)
			state.Results[fingerprint] = name
			if err := saveSnapshotAnchorState(statePath, *state); err != nil {
				return snapshotResult{}, true, err
			}
			return snapshotResult{CodeSourcePath: path.Join(uploadPath, name)}, true, nil
		}
	}

	log.Debugf(ctx, "air snapshot overlay: decision=new-anchor branch=%q reason=%s files=%d uncompressed_bytes=%d", branch, anchorReason, len(files), totalBytes)
	archivePath := filepath.Join(tmp, "anchor.tar.gz")
	packageStart := time.Now()
	if err := packageSnapshot(ctx, repoPath, configPath, plan, files, archivePath, false); err != nil {
		return snapshotResult{}, true, err
	}
	afterFiles, err := snapshotFiles(ctx, repoPath, plan.includePaths, plan.isGitRepo)
	if err != nil {
		return snapshotResult{}, true, err
	}
	if snapshotOverlayFingerprint(afterFiles) != fingerprint {
		return snapshotResult{}, true, errors.New("code source changed while creating its anchor snapshot; retry submission")
	}
	packageDuration := time.Since(packageStart)
	archiveInfo, err := os.Stat(archivePath)
	if err != nil {
		return snapshotResult{}, true, err
	}
	publishStart := time.Now()
	name, err := publishImmutableSnapshot(ctx, store, "air_anchor_"+snapshotOverlayVersion, archivePath)
	if err != nil {
		return snapshotResult{}, true, err
	}
	log.Debugf(ctx, "air snapshot overlay: decision=anchor package=%s publish=%s compressed_bytes=%d object=%s",
		packageDuration, time.Since(publishStart), archiveInfo.Size(), name)
	newState := snapshotAnchorState{
		Version:           snapshotOverlayVersion,
		Branch:            branch,
		CreatedAtUnix:     time.Now().Unix(),
		AnchorName:        name,
		AnchorFingerprint: fingerprint,
		Component:         component,
		Entries:           snapshotOverlayManifest(files),
		Results:           map[string]string{fingerprint: name},
	}
	if err := saveSnapshotAnchorState(statePath, newState); err != nil {
		return snapshotResult{}, true, err
	}
	return snapshotResult{CodeSourcePath: path.Join(uploadPath, name)}, true, nil
}

func createSnapshotOverlay(
	repoPath string,
	component string,
	state *snapshotAnchorState,
	resultFingerprint string,
	diff snapshotOverlayDiff,
	outputPath string,
) (err error) {
	output, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create overlay: %w", err)
	}
	gz, err := pgzip.NewWriterLevel(output, pgzip.DefaultCompression)
	if err != nil {
		output.Close()
		return err
	}
	tw := tar.NewWriter(gz)
	defer func() {
		err = firstErr(err, tw.Close(), gz.Close(), output.Close())
	}()

	descriptor, err := json.Marshal(snapshotOverlayDescriptor{
		FormatVersion:     1,
		Anchor:            state.AnchorName,
		AnchorFingerprint: state.AnchorFingerprint,
		ResultFingerprint: resultFingerprint,
		Component:         component,
	})
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{
		Name: snapshotOverlayDescriptorName,
		Mode: 0o600,
		Size: int64(len(descriptor)),
	}); err != nil {
		return fmt.Errorf("failed to write overlay descriptor: %w", err)
	}
	if _, err := tw.Write(descriptor); err != nil {
		return fmt.Errorf("failed to write overlay descriptor: %w", err)
	}

	for _, rel := range diff.Deleted {
		if err := validateSnapshotRelativePath(rel); err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: path.Join(snapshotOverlayDeletionsRoot, rel),
			Mode: 0o600,
		}); err != nil {
			return fmt.Errorf("failed to write deletion marker for %q: %w", rel, err)
		}
	}
	for _, file := range diff.Changed {
		if err := writeSnapshotOverlayMember(tw, repoPath, component, file); err != nil {
			return err
		}
	}
	return nil
}

func validateSnapshotRelativePath(rel string) error {
	clean := path.Clean(filepath.ToSlash(rel))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) || clean != filepath.ToSlash(rel) {
		return fmt.Errorf("invalid snapshot path %q", rel)
	}
	return nil
}

func writeSnapshotOverlayMember(tw *tar.Writer, repoPath, component string, file snapshotFile) error {
	rel := filepath.ToSlash(file.rel)
	if err := validateSnapshotRelativePath(rel); err != nil {
		return err
	}
	fullPath := filepath.Join(repoPath, file.rel)
	info, linkTarget, err := inspectSnapshotOverlayFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to inspect changed snapshot file %q: %w", rel, err)
	}
	if !snapshotFileMatches(file, info, linkTarget) {
		return fmt.Errorf("snapshot file changed while packaging: %q", rel)
	}
	if !info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	header, err := tar.FileInfoHeader(info, linkTarget)
	if err != nil {
		return fmt.Errorf("failed to build overlay header for %q: %w", rel, err)
	}
	header.Name = path.Join(component, rel)
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write overlay header for %q: %w", rel, err)
	}
	if info.Mode().IsRegular() {
		input, err := os.Open(fullPath)
		if err != nil {
			return fmt.Errorf("failed to open changed snapshot file %q: %w", rel, err)
		}
		_, copyErr := io.Copy(tw, input)
		closeErr := input.Close()
		if copyErr != nil {
			return fmt.Errorf("failed to write changed snapshot file %q: %w", rel, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("failed to close changed snapshot file %q: %w", rel, closeErr)
		}
	}
	after, afterLink, err := inspectSnapshotOverlayFile(fullPath)
	if err != nil || !snapshotFileMatches(file, after, afterLink) {
		return fmt.Errorf("snapshot file changed while packaging: %q", rel)
	}
	return nil
}

func inspectSnapshotOverlayFile(name string) (os.FileInfo, string, error) {
	info, err := os.Lstat(name)
	if err != nil {
		return nil, "", err
	}
	linkTarget := ""
	if info.Mode()&os.ModeSymlink != 0 {
		linkTarget, err = os.Readlink(name)
	}
	return info, linkTarget, err
}

func snapshotFileMatches(file snapshotFile, info os.FileInfo, linkTarget string) bool {
	return info != nil && file.size == info.Size() && file.modTime == info.ModTime().UnixNano() &&
		file.mode == uint32(info.Mode()) && file.linkTarget == linkTarget
}

func publishImmutableSnapshot(ctx context.Context, store filer.Filer, kind, archivePath string) (string, error) {
	input, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		input.Close()
		return "", fmt.Errorf("failed to hash snapshot archive: %w", err)
	}
	if _, err := input.Seek(0, io.SeekStart); err != nil {
		input.Close()
		return "", err
	}
	name := kind + "_" + hex.EncodeToString(hash.Sum(nil)) + ".tar.gz"
	err = store.Write(ctx, name, input, filer.CreateParentDirectories)
	closeErr := input.Close()
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return "", fmt.Errorf("failed to publish immutable snapshot %s: %w", name, err)
	}
	if closeErr != nil {
		return "", closeErr
	}
	localInfo, err := os.Stat(archivePath)
	if err != nil {
		return "", err
	}
	remoteInfo, err := store.Stat(ctx, name)
	if err != nil {
		return "", fmt.Errorf("failed to verify immutable snapshot %s: %w", name, err)
	}
	if remoteInfo.Size() != localInfo.Size() {
		return "", fmt.Errorf("immutable snapshot %s has unexpected size: got %d, want %d", name, remoteInfo.Size(), localInfo.Size())
	}
	return name, nil
}
