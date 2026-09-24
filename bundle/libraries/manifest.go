package libraries

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/patchwheel"
)

// NewUploadManifest records the local files in libs using portable paths.
func NewUploadManifest(ctx context.Context, b *bundle.Bundle, libs map[string][]LocationToUpdate) ([]deployplan.UploadManifestEntry, error) {
	_ = ctx
	original := make(map[string]string)
	for _, artifact := range b.Config.Artifacts {
		if artifact == nil {
			continue
		}
		for _, file := range artifact.Files {
			if file.Patched != "" {
				original[file.Patched] = file.Source
			}
		}
	}

	entries := make([]deployplan.UploadManifestEntry, 0, len(libs))
	sources := mapsKeys(libs)
	slices.Sort(sources)
	for _, source := range sources {
		manifestSource, patched := original[source]
		if manifestSource == "" {
			manifestSource = source
		}
		if !filepath.IsAbs(manifestSource) {
			manifestSource = filepath.Join(b.SyncRootPath, manifestSource)
		}
		rel, err := filepath.Rel(b.BundleRootPath, manifestSource)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return nil, fmt.Errorf("upload source %q is outside bundle root", manifestSource)
		}
		digest, err := fileSHA256(source)
		if err != nil {
			return nil, fmt.Errorf("hashing upload source %q: %w", source, err)
		}
		entries = append(entries, deployplan.UploadManifestEntry{
			Source:       filepath.ToSlash(rel),
			RemoteName:   filepath.Base(source),
			SHA256:       digest,
			PatchedWheel: patched,
		})
	}

	return entries, nil
}

// PrepareUploadManifest validates a saved plan's upload manifest and returns the
// local paths to upload. The returned cleanup function removes temporary patched
// wheels created during preflight.
func PrepareUploadManifest(ctx context.Context, b *bundle.Bundle, manifest []deployplan.UploadManifestEntry) (map[string][]LocationToUpdate, func(), error) {
	libs := make(map[string][]LocationToUpdate, len(manifest))
	var temporary []string
	cleanup := func() {
		for _, path := range temporary {
			_ = os.RemoveAll(path)
		}
	}

	for _, entry := range manifest {
		if strings.ContainsAny(entry.Source, "*?[") {
			cleanup()
			return nil, func() {}, fmt.Errorf("upload manifest source %q contains a glob", entry.Source)
		}
		if filepath.IsAbs(entry.Source) {
			cleanup()
			return nil, func() {}, fmt.Errorf("upload manifest source %q is not relative", entry.Source)
		}
		source := filepath.Join(b.BundleRootPath, filepath.FromSlash(entry.Source))
		if _, err := os.Stat(source); err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("reading upload manifest source %q: %w", entry.Source, err)
		}

		if !entry.PatchedWheel {
			if filepath.Base(source) != entry.RemoteName {
				cleanup()
				return nil, func() {}, fmt.Errorf("upload manifest source %q has basename %q, expected %q", entry.Source, filepath.Base(source), entry.RemoteName)
			}
			if err := verifyDigest(source, entry.SHA256); err != nil {
				cleanup()
				return nil, func() {}, fmt.Errorf("validating upload manifest source %q: %w", entry.Source, err)
			}
			libs[source] = nil
			continue
		}

		cacheDir := filepath.Join(b.GetLocalStateDir(ctx), "patched_wheels")
		cached, err := findCachedWheel(cacheDir, entry.RemoteName, entry.SHA256)
		if err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("checking cached upload %q: %w", entry.RemoteName, err)
		}
		if cached == "" {
			temporaryDir, err := os.MkdirTemp("", "databricks-cli-patched-wheel-")
			if err != nil {
				cleanup()
				return nil, func() {}, fmt.Errorf("creating temporary wheel directory: %w", err)
			}
			temporary = append(temporary, temporaryDir)
			cached, _, err = patchwheel.PatchWheel(source, temporaryDir)
			if err != nil {
				cleanup()
				return nil, func() {}, fmt.Errorf("patching upload manifest wheel %q: %w", entry.Source, err)
			}
			if err := verifyDigest(cached, entry.SHA256); err != nil {
				cleanup()
				return nil, func() {}, fmt.Errorf("validating patched upload manifest wheel %q: %w", entry.Source, err)
			}
		}
		if filepath.Base(cached) != entry.RemoteName {
			cleanup()
			return nil, func() {}, fmt.Errorf("patched upload manifest wheel %q has basename %q, expected %q", entry.Source, filepath.Base(cached), entry.RemoteName)
		}
		libs[cached] = nil
	}
	return libs, cleanup, nil
}

func mapsKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func fileSHA256(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifyDigest(name, expected string) error {
	actual, err := fileSHA256(name)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("sha256 mismatch: got %s, expected %s", actual, expected)
	}
	return nil
}

func findCachedWheel(dir, name, digest string) (string, error) {
	var match string
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if entry.IsDir() || filepath.Base(path) != name {
			return nil
		}
		if err := verifyDigest(path, digest); err == nil {
			match = path
			return filepath.SkipAll
		}
		return nil
	})
	return match, err
}
