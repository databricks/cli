package dbconnect

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/databricks/cli/libs/log"
)

// Constraints holds the parsed contents of a per-environment pyproject.toml.
type Constraints struct {
	// EnvKey is the environment key used to look up the constraints.
	EnvKey string
	// SourceURL is the URL from which the constraints were fetched.
	SourceURL string
	// FromCache is true when the data came from the on-disk cache rather than a live fetch.
	FromCache bool
	// RequiresPython is the PEP 440 python version specifier from [project].requires-python.
	RequiresPython string
	// DatabricksConnect is the full dependency string for databricks-connect from [dependency-groups].dev.
	DatabricksConnect string
	// ConstraintDeps is the list of entries from [tool.uv].constraint-dependencies.
	ConstraintDeps []string
}

// sanitizeEnvKey replaces path separators with double-underscores to produce a flat filename.
func sanitizeEnvKey(envKey string) string {
	return strings.ReplaceAll(envKey, "/", "__")
}

// FetchConstraints fetches the pyproject.toml for envKey from baseURL, caches it in cacheDir,
// and falls back to the cached copy on network or HTTP errors.
//
// Constraint files are hosted at:
// https://github.com/pietern/databricks-environments
func FetchConstraints(ctx context.Context, baseURL, envKey, cacheDir string) (*Constraints, error) {
	url := baseURL + "/" + envKey + "/pyproject.toml"
	cachePath := filepath.Join(cacheDir, sanitizeEnvKey(envKey)+".toml")

	data, fetchErr := fetchURL(ctx, url)
	if fetchErr == nil {
		// Write the cache copy; non-fatal so a read-only cacheDir doesn't break the command.
		if err := os.WriteFile(cachePath, data, 0o600); err != nil {
			log.Debugf(ctx, "failed to write constraint cache %s: %v", filepath.ToSlash(cachePath), err)
		}
		rp, dbc, deps, err := parseConstraints(data)
		if err != nil {
			return nil, fmt.Errorf("parse constraints for %s: %w", envKey, err)
		}
		return &Constraints{
			EnvKey:            envKey,
			SourceURL:         url,
			FromCache:         false,
			RequiresPython:    rp,
			DatabricksConnect: dbc,
			ConstraintDeps:    deps,
		}, nil
	}

	// Network or HTTP failure: attempt to serve from cache.
	cached, readErr := os.ReadFile(cachePath)
	if readErr != nil {
		return nil, NewError(ErrConstraintFetchFailed, fetchErr, "fetch constraints for %s", envKey)
	}

	log.Warnf(ctx, "constraint fetch failed, using cached copy: %v", fetchErr)
	rp, dbc, deps, err := parseConstraints(cached)
	if err != nil {
		return nil, fmt.Errorf("parse cached constraints for %s: %w", envKey, err)
	}
	return &Constraints{
		EnvKey:            envKey,
		SourceURL:         url,
		FromCache:         true,
		RequiresPython:    rp,
		DatabricksConnect: dbc,
		ConstraintDeps:    deps,
	}, nil
}

// fetchURL performs an HTTP GET and returns the body bytes, or an error on non-2xx or transport failure.
func fetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", url, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body from %s: %w", url, err)
	}
	return data, nil
}

// pyprojectTOML mirrors the pyproject.toml fields we care about.
type pyprojectTOML struct {
	Project struct {
		RequiresPython string `toml:"requires-python"`
	} `toml:"project"`
	DependencyGroups struct {
		Dev []string `toml:"dev"`
	} `toml:"dependency-groups"`
	Tool struct {
		UV struct {
			ConstraintDependencies []string `toml:"constraint-dependencies"`
		} `toml:"uv"`
	} `toml:"tool"`
}

// parseConstraints parses a pyproject.toml byte slice and extracts requires-python,
// the databricks-connect entry from dependency-groups.dev, and constraint-dependencies.
func parseConstraints(data []byte) (requiresPython, dbconnect string, deps []string, err error) {
	var p pyprojectTOML
	if err = toml.Unmarshal(data, &p); err != nil {
		return "", "", nil, fmt.Errorf("unmarshal pyproject.toml: %w", err)
	}

	requiresPython = p.Project.RequiresPython

	for _, entry := range p.DependencyGroups.Dev {
		// Despace before matching so whitespace variants like "databricks-connect ~=17" also match.
		if strings.HasPrefix(strings.ReplaceAll(entry, " ", ""), "databricks-connect") {
			dbconnect = entry
			break
		}
	}

	deps = p.Tool.UV.ConstraintDependencies
	return requiresPython, dbconnect, deps, nil
}
