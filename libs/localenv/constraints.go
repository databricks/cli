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
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

// EnvConstraintSourceURLTestOverride names the environment variable that
// overrides the constraint source with a full base URL. It exists so tests can
// point the fetch at a local server (see the acceptance suite) and for power-user
// debugging; normal runs use defaultConstraintBaseURL. The TEST_OVERRIDE naming
// signals it is not a supported user-facing knob.
const EnvConstraintSourceURLTestOverride = "DATABRICKS_LOCALENV_CONSTRAINT_SOURCE_URL_TEST_OVERRIDE"

// defaultConstraintBaseURL is the base URL of the published constraint artifacts.
//
// The databricks/environments repo nests its language ecosystems under a
// top-level directory, so the Python artifacts live at python/<env key>/
// pyproject.toml (e.g. python/serverless/serverless-v5, python/dbr/<spark>),
// not at the repo root. The base URL is anchored at that python/ subtree so an
// env key of "serverless/serverless-v5" resolves to the real path.
const defaultConstraintBaseURL = "https://raw.githubusercontent.com/databricks/environments/main/python"

// ConstraintBaseURL returns the base URL for constraint artifacts: the
// EnvConstraintSourceURLTestOverride value when set (a full base URL), otherwise
// the built-in defaultConstraintBaseURL.
func ConstraintBaseURL(ctx context.Context) string {
	if v, ok := env.Lookup(ctx, EnvConstraintSourceURLTestOverride); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return defaultConstraintBaseURL
}

// errEnvKeyNotFound is returned by fetchURL when the constraint artifact does
// not exist for the requested env key (HTTP 404). It is distinct from a
// transport failure so FetchConstraints can classify it as E_ENV_UNSUPPORTED
// (a resolvable target with no published environment) rather than E_FETCH.
var errEnvKeyNotFound = errors.New("environment key not found")

// maxConstraintBytes caps the constraint artifact read. The body is untrusted
// remote content and a pyproject.toml is small; 1 MiB is far above any real
// artifact while preventing a misbehaving or hostile host from exhausting memory.
const maxConstraintBytes int64 = 1 << 20

// constraintHTTPClient fetches constraint artifacts with an explicit timeout, so
// the request is bounded even when the caller's context has no deadline.
var constraintHTTPClient = &http.Client{Timeout: 30 * time.Second}

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
	// EnvironmentVersion is the serverless environment version written into
	// [tool.databricks.environment].environment_version (e.g. "5"). It is not
	// parsed from the artifact but set by the pipeline from the resolved compute
	// target; it is empty for cluster targets, where the section is not managed.
	EnvironmentVersion string
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
// baseURL points at the base of the constraint artifacts (see ConstraintBaseURL).
//
// writeCache controls whether a successful live fetch populates the on-disk
// cache. Callers pass false for a dry run (--dry-run), which must not mutate
// disk; an existing cache is still read for offline fallback, since reading is
// not a mutation.
func FetchConstraints(ctx context.Context, baseURL, envKey, cacheDir string, writeCache bool) (*Constraints, error) {
	if baseURL == "" {
		// ConstraintBaseURL always returns a non-empty default, so an empty baseURL
		// here means the resolver was bypassed. Report it at the fetch phase so it
		// still flows through the same phase/JSON reporting as any other fetch error.
		return nil, NewError(ErrFetch, nil, "no constraint source configured")
	}
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
		// so a read-only cacheDir doesn't break the command. Skipped under a dry
		// run so --dry-run performs no disk writes at all.
		if writeCache {
			if err := writeCacheAtomic(cachePath, data); err != nil {
				log.Debugf(ctx, "failed to write constraint cache %s: %v", filepath.ToSlash(cachePath), err)
			}
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
			"no published environment for %q. If this is a new runtime, try the latest LTS target (e.g. --serverless-version 5 or a supported --cluster-id DBR)", envKey)
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
	// Use a client with an explicit timeout rather than http.DefaultClient, which
	// has none: the fetch of remote content must be bounded even if the caller's
	// context carries no deadline.
	resp, err := constraintHTTPClient.Do(req)
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
	// Cap the read: the body is untrusted remote content and a pyproject.toml is
	// small, so an oversized (or hostile) response must not be read into memory
	// unbounded. Read one byte past the cap to detect an over-limit body.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxConstraintBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read body from %s: %w", url, err)
	}
	if int64(len(data)) > maxConstraintBytes {
		return nil, fmt.Errorf("constraint artifact from %s exceeds %d bytes", url, maxConstraintBytes)
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
	// Reject a requires-python that is present but yields no installable floor
	// (e.g. ">=3", "<3.13", "*") here, before the body is cached. This is the
	// same check the pipeline applies later; running it in the pre-cache guard
	// ensures an unusable 2xx body cannot overwrite a valid cached copy and
	// break offline fallback, which is the guard's stated purpose.
	if _, err = PythonMinorFromRequires(requiresPython); err != nil {
		return "", "", nil, fmt.Errorf("constraint artifact has an unusable [project].requires-python: %w", err)
	}

	for _, entry := range p.DependencyGroups.Dev {
		if isDatabricksConnectDep(entry) {
			dbconnect = entry
			break
		}
	}

	// Normalize a missing [tool.uv].constraint-dependencies to a non-nil empty
	// slice. A nil ConstraintDeps is reserved as the --no-constraints "leave the
	// constraint block unmanaged" signal (mergeToolUv and RenderFreshPyproject skip
	// on nil); without this, an artifact that simply omits the key would be
	// indistinguishable from the flag and would silently stop being managed.
	deps = p.Tool.UV.ConstraintDependencies
	if deps == nil {
		deps = []string{}
	}
	return requiresPython, dbconnect, deps, nil
}

// depNameSepRe matches the first PEP 508 delimiter that ends a requirement's
// package name: a version specifier, extra, marker, url, or list separator.
var depNameSepRe = regexp.MustCompile(`[<>=!~;,@\[( \t]`)

// isDepNamed reports whether a dependency-group entry names the package normalizedName
// (which the caller passes already PEP 503-normalized). It extracts the leading package
// name (up to the first PEP 508 delimiter) and compares it under PEP 503 normalization,
// so case and runs of "-", "_", or "." are all treated as equivalent: for
// "databricks-connect", "Databricks-Connect", "databricks_connect", and
// "databricks.connect" all match, while a distinct package like "databricks-connectors"
// does not.
func isDepNamed(entry, normalizedName string) bool {
	name := strings.TrimSpace(entry)
	if i := depNameSepRe.FindStringIndex(name); i != nil {
		name = name[:i[0]]
	}
	return normalizePackageName(name) == normalizedName
}

// isDatabricksConnectDep reports whether a dependency-group entry is the
// databricks-connect requirement.
func isDatabricksConnectDep(entry string) bool { return isDepNamed(entry, "databricks-connect") }

// isPysparkDep reports whether a dependency-group entry is the standalone pyspark
// requirement. It matches "pyspark" exactly and not a distinct package such as
// "pyspark-stubs".
func isPysparkDep(entry string) bool { return isDepNamed(entry, "pyspark") }

// pep503SepRe matches runs of "-", "_", or "." for PEP 503 name normalization.
var pep503SepRe = regexp.MustCompile(`[-_.]+`)

// normalizePackageName applies PEP 503 normalization: lowercase and collapse any
// run of "-", "_", or "." to a single "-".
func normalizePackageName(name string) string {
	return pep503SepRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
}
