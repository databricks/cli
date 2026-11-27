package skills

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// releaseTimeout bounds one call to the published release, the release lookup
// included, so an interactive command falls back to another source promptly
// when GitHub does not answer. A variable so the tests can shorten it.
var releaseTimeout = 5 * time.Second

type releaseSource struct{}

// NewReleaseSource returns a Source over the published skills release: the
// release GetSkillsRef resolves, read from GetSkillsBaseURL. Each call reads
// the release again. A release that cannot be reached, by a timeout, a refused
// connection or a server error, fails with ErrSourceUnreachable.
func NewReleaseSource() Source {
	return releaseSource{}
}

func (releaseSource) Catalog(ctx context.Context) (*Catalog, error) {
	ctx, cancel := context.WithTimeout(ctx, releaseTimeout)
	defer cancel()
	src, err := openRelease(ctx)
	if err != nil {
		return nil, err
	}
	return src.Catalog(ctx)
}

func (releaseSource) ReadFile(ctx context.Context, skill, file string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, releaseTimeout)
	defer cancel()
	src, err := openRelease(ctx)
	if err != nil {
		return nil, err
	}
	return src.ReadFile(ctx, skill, file)
}

// openRelease resolves the release ref with GetSkillsRef and returns a Source
// that reads that release's files over HTTP. It does not use the installer's
// embedded-version fallback: a failed lookup is ErrSourceUnreachable, and the
// caller decides what to serve instead.
func openRelease(ctx context.Context) (Source, error) {
	ref, _, err := GetSkillsRef(ctx)
	if err != nil {
		return nil, unreachable(err)
	}
	base := GetSkillsBaseURL(ctx) + "/" + ref
	return &fsSource{
		read: func(ctx context.Context, name string) ([]byte, error) {
			// Only the manifest is capped, as the installer caps it. Skill
			// files are read whole.
			return fetchRelease(ctx, releaseFileURL(base, name), name == manifestFile)
		},
		label: base,
	}, nil
}

// releaseFileURL returns the URL of the file name in the release at base. Each
// path segment is escaped, so a name with '#', '?' or '%' stays part of the
// path instead of starting a fragment or a query.
func releaseFileURL(base, name string) string {
	segments := strings.Split(name, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return base + "/" + strings.Join(segments, "/")
}

// unreachable marks err as ErrSourceUnreachable. It leaves err alone when err
// already carries the mark, so an error is not wrapped twice, and when the caller
// cancelled the call: a cancel is the caller's choice, not a source that cannot
// be reached, so a fallback caller must not try another source after it.
func unreachable(err error) error {
	if errors.Is(err, ErrSourceUnreachable) || errors.Is(err, context.Canceled) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrSourceUnreachable, err)
}

// errReleaseNotFound reports a release file the server does not have (HTTP 404).
// It matches fs.ErrNotExist and not ErrSourceUnreachable, so a caller that falls
// back on unreachable sources does not serve other content for a missing file
// or for a ref that names no release. The error names the file's URL, which
// carries the ref.
var errReleaseNotFound = errors.New("release file not found")

// fetchRelease downloads one file of a release. A 404 is errReleaseNotFound.
// Every other failure to get the file is ErrSourceUnreachable, except a capped
// read of a body over MaxFetchBodySize, which is ErrManifestTooLarge. The capped
// read stops one byte past the cap, so an oversized body is told apart from one
// that fits exactly.
func fetchRelease(ctx context.Context, fileURL string, capped bool) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, unreachable(err)
	}
	resp, err := HTTPClient().Do(req)
	if err != nil {
		return nil, unreachable(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w at %s: %w", errReleaseNotFound, fileURL, fs.ErrNotExist)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d from %s", ErrSourceUnreachable, resp.StatusCode, fileURL)
	}
	var body io.Reader = resp.Body
	if capped {
		body = io.LimitReader(resp.Body, MaxFetchBodySize+1)
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, unreachable(err)
	}
	if capped && len(data) > MaxFetchBodySize {
		return nil, ErrManifestTooLarge
	}
	return data, nil
}
