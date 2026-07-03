package localenv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/databricks/cli/libs/log"
)

// errEnvKeyNotFound is returned by fetchURL when the constraint artifact does
// not exist for the requested env key (HTTP 404). It is distinct from a
// transport failure so FetchConstraints can classify it as E_ENV_UNSUPPORTED
// (a resolvable target with no published environment) rather than E_FETCH.
var errEnvKeyNotFound = errors.New("environment key not found")

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

// cacheFileName maps an env key to a single, collision-free cache filename.
// It keeps a readable slug (path separators flattened to double-underscores so
// the file stays inside cacheDir on every OS) and appends a short hash of the
// raw env key. The hash guarantees injectivity: distinct env keys that would
// otherwise flatten to the same slug (e.g. "a/b" and "a__b") get distinct
// filenames, so a cache hit can never serve another environment's constraints.
func cacheFileName(envKey string) string {
	slug := strings.ReplaceAll(envKey, "/", "__")
	slug = strings.ReplaceAll(slug, "\\", "__")
	sum := sha256.Sum256([]byte(envKey))
	return fmt.Sprintf("%s-%s.toml", slug, hex.EncodeToString(sum[:8]))
}

// writeCacheAtomic writes data to path via a temp file and rename, creating the
// parent directory first. The rename is atomic on the same filesystem, so a
// concurrent reader never observes a truncated or partial cache file (os.WriteFile
// truncates in place, which a fallback reader could catch mid-write).
func writeCacheAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".constraints-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// FetchConstraints fetches the pyproject.toml for envKey from baseURL and caches it in
// cacheDir. On a transport or non-404 HTTP failure it falls back to the cached copy if one
// exists (E_FETCH otherwise). A 404 means the env key is not published (E_ENV_UNSUPPORTED)
// and does not fall back to cache — a resolvable target with no environment is a distinct,
// non-transient condition.
//
// Constraint files are hosted at:
// https://github.com/rugpanov/databricks-environments
func FetchConstraints(ctx context.Context, baseURL, envKey, cacheDir string) (*Constraints, error) {
	url := baseURL + "/" + envKey + "/pyproject.toml"
	cachePath := filepath.Join(cacheDir, cacheFileName(envKey))

	data, fetchErr := fetchURL(ctx, url)
	if fetchErr == nil {
		// Parse before caching: a malformed 2xx body must not overwrite a valid
		// cached copy, or a later transport-failure run would serve the poisoned
		// cache and fail to parse instead of falling back to the last-good file.
		rp, dbc, deps, err := parseConstraints(data)
		if err != nil {
			return nil, fmt.Errorf("parse constraints for %s: %w", envKey, err)
		}
		// Write the cache copy (creating cacheDir if needed, atomically); non-fatal
		// so a read-only cacheDir doesn't break the command.
		if err := writeCacheAtomic(cachePath, data); err != nil {
			log.Debugf(ctx, "failed to write constraint cache %s: %v", filepath.ToSlash(cachePath), err)
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

	// A missing env key (404) is not a transport failure and has no useful cache
	// fallback: the target resolved to an environment that isn't published.
	if errors.Is(fetchErr, errEnvKeyNotFound) {
		return nil, NewError(ErrEnvUnsupported, fetchErr,
			"no published environment for %q. If this is a new runtime, try the latest LTS target (e.g. --serverless v4 or a supported --cluster DBR)", envKey)
	}

	// Network or HTTP failure: attempt to serve from cache.
	cached, readErr := os.ReadFile(cachePath)
	if readErr != nil {
		return nil, NewError(ErrFetch, fetchErr, "fetch constraints for %s", envKey)
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
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("GET %s: %w", url, errEnvKeyNotFound)
	}
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
// A body that is valid TOML but carries no requires-python is rejected: it is not a
// usable constraint artifact, and silently accepting it would cache an empty result
// and only surface a confusing failure later in the pipeline.
func parseConstraints(data []byte) (requiresPython, dbconnect string, deps []string, err error) {
	var p pyprojectTOML
	if err = toml.Unmarshal(data, &p); err != nil {
		return "", "", nil, fmt.Errorf("unmarshal pyproject.toml: %w", err)
	}

	requiresPython = p.Project.RequiresPython
	if strings.TrimSpace(requiresPython) == "" {
		return "", "", nil, errors.New("constraint artifact has no [project].requires-python")
	}

	for _, entry := range p.DependencyGroups.Dev {
		if isDatabricksConnectDep(entry) {
			dbconnect = entry
			break
		}
	}

	deps = p.Tool.UV.ConstraintDependencies
	return requiresPython, dbconnect, deps, nil
}

// isDatabricksConnectDep reports whether a dependency-group entry is the
// databricks-connect requirement. It matches on a package-name boundary rather
// than a bare prefix so a sibling package such as "databricks-connectors" (whose
// name merely starts with "databricks-connect") is not mistaken for it. The next
// character after the name must be a PEP 508 version/extra/marker delimiter or the
// end of the string.
func isDatabricksConnectDep(entry string) bool {
	const name = "databricks-connect"
	// Despace so whitespace variants like "databricks-connect ~=17" also match.
	s := strings.ReplaceAll(entry, " ", "")
	rest, ok := strings.CutPrefix(s, name)
	if !ok {
		return false
	}
	if rest == "" {
		return true
	}
	// A real requirement continues with a version specifier, extra, marker, or
	// separator — never an identifier character (which would mean a longer name).
	switch rest[0] {
	case '=', '<', '>', '!', '~', '[', ';', '@', ',', '(':
		return true
	default:
		return false
	}
}
