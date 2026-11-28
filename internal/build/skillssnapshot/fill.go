package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/skills"
	"golang.org/x/sync/errgroup"
)

// fetchConcurrency bounds the parallel file downloads.
const fetchConcurrency = 8

var httpClient = &http.Client{Timeout: 60 * time.Second}

// fillSnapshot replaces dir with the files of the skills release ref, or of the
// latest release when ref is empty, and returns the release it bundled. The
// manifest is checked as every source checks it before anything is written. Any
// failed download or check leaves dir as it was, so a release build cannot ship
// a partial snapshot.
func fillSnapshot(ctx context.Context, dir, ref string) (string, error) {
	if ref == "" {
		var err error
		ref, err = skills.FetchLatestSkillsReleaseRef(ctx)
		if err != nil {
			return "", err
		}
	}
	base := skills.GetSkillsBaseURL(ctx) + "/" + ref

	raw, err := get(ctx, base+"/manifest.json")
	if err != nil {
		return "", fmt.Errorf("fetch skills manifest at %s: %w", ref, err)
	}
	var m skills.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", fmt.Errorf("parse skills manifest at %s: %w", ref, err)
	}
	skills.NormalizeManifest(&m)
	if err := skills.CheckManifest(&m); err != nil {
		return "", err
	}
	files, err := releaseFiles(&m)
	if err != nil {
		return "", err
	}

	stage, err := os.MkdirTemp(filepath.Dir(dir), "."+filepath.Base(dir)+"-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)

	if err := os.WriteFile(filepath.Join(stage, "manifest.json"), raw, 0o644); err != nil {
		return "", err
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(fetchConcurrency)
	for _, rel := range files {
		g.Go(func() error { return fetchTo(gctx, base, stage, rel) })
	}
	if err := g.Wait(); err != nil {
		return "", err
	}

	// Read the staged release the way the bundled source will at run time.
	if _, err := skills.NewFSSource(os.DirFS(stage), skills.BundledLabel).Catalog(ctx); err != nil {
		return "", err
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	if err := os.Rename(stage, dir); err != nil {
		return "", err
	}
	return ref, nil
}

// releaseFiles lists the files a snapshot needs, relative to the release root:
// the catalog rules and every skill file. go:embed skips a file with an element
// starting with "." or "_", so the binary would lack such a file. It is refused
// here instead.
func releaseFiles(m *skills.Manifest) ([]string, error) {
	var files []string
	if m.CatalogRules != "" {
		files = append(files, path.Clean(m.CatalogRules))
	}
	for _, name := range slices.Sorted(maps.Keys(m.Skills)) {
		meta := m.Skills[name]
		for _, file := range meta.Files {
			files = append(files, path.Join(meta.RepoDir, name, file))
		}
	}
	for _, rel := range files {
		for elem := range strings.SplitSeq(rel, "/") {
			if strings.HasPrefix(elem, ".") || strings.HasPrefix(elem, "_") {
				return nil, fmt.Errorf("release file %q has element %q, which go:embed skips", rel, elem)
			}
		}
	}
	return files, nil
}

// fetchTo downloads the release file rel into the staging folder.
func fetchTo(ctx context.Context, base, stage, rel string) error {
	data, err := get(ctx, base+"/"+rel)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", rel, err)
	}
	dest := filepath.Join(stage, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

// get returns the body of a GET that must answer 200.
func get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, rawURL)
	}
	return io.ReadAll(resp.Body)
}
