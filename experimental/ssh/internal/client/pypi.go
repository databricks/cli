package client

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/localenv"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/net/html"
)

// defaultPyPIIndexURL is the index pip uses when none is configured.
const defaultPyPIIndexURL = "https://pypi.org/simple"

// maxIndexPageSize and maxWheelSize bound what is read into memory from the index. The
// databricks project page is about 20 KB and a CLI wheel about 20 MB.
const (
	maxIndexPageSize = 16 << 20
	maxWheelSize     = 256 << 20
)

// pypiHTTPClient bounds each request, which http.DefaultClient does not: a proxy that
// accepts the connection and then stalls must not hang `ssh connect`. The limit leaves
// room to download a wheel over a slow link.
var pypiHTTPClient = &http.Client{Timeout: 10 * time.Minute, CheckRedirect: checkPyPIRedirect}

// checkPyPIRedirect keeps the index credentials on the index origin, the exact scheme and
// host of the index URL, across redirects: net/http sends the userinfo of a URL only with
// requests to that URL, so an absolute redirect back to the index would go without it.
// Redirects elsewhere get neither the userinfo nor an Authorization header, which net/http
// keeps on redirects to subdomains. It also drops the Referer that net/http sets to the
// previous URL, whose query string can carry an index token.
func checkPyPIRedirect(req *http.Request, via []*http.Request) error {
	// The limit of the default policy, which this one replaces.
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	req.Header.Del("Referer")
	first := via[0].URL
	if first.User != nil && first.Scheme == "https" && req.URL.Scheme == "http" {
		return fmt.Errorf("refusing to follow a redirect from HTTPS to plain HTTP for an index URL with credentials, from %s to %s", first.Host, req.URL.Host)
	}
	if req.URL.Scheme != first.Scheme || req.URL.Host != first.Host {
		req.Header.Del("Authorization")
		req.URL.User = nil
		return nil
	}
	if req.URL.User == nil {
		req.URL.User = first.User
	}
	return nil
}

// wheelMachines maps a release architecture to the machine in the Linux wheel platform tags.
var wheelMachines = map[string]string{
	"amd64": "x86_64",
	"arm64": "aarch64",
}

// getPyPIRelease builds the release archive for linux/<architecture> from the CLI's wheel
// on the package index that pypiIndexURL returns.
func getPyPIRelease(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
	indexURL, err := url.Parse(pypiIndexURL(ctx))
	if err != nil {
		// The parse error repeats the URL, which can carry credentials.
		return nil, errors.New("the Python package index URL is not a valid URL")
	}
	wheelURL, wheelSHA256, err := findLinuxWheel(ctx, indexURL, architecture, version)
	if err != nil {
		return nil, err
	}
	wheel, err := downloadWheel(ctx, wheelURL, wheelSHA256)
	if err != nil {
		return nil, err
	}
	archive, err := releaseArchiveFromWheel(wheel, version)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(archive)), nil
}

// pypiIndexURL returns the package index to download the CLI wheels from: PIP_INDEX_URL,
// else index-url in the per-user pip config, else PyPI. Networks that block PyPI point pip
// at a mirror of it there. System-wide pip config files are not read.
func pypiIndexURL(ctx context.Context) string {
	// https://pip.pypa.io/en/stable/topics/configuration/#environment-variables
	if indexURL := env.Get(ctx, "PIP_INDEX_URL"); indexURL != "" {
		return indexURL
	}
	if indexURL := localenv.PipConfIndexURL(ctx); indexURL != "" {
		return indexURL
	}
	return defaultPyPIIndexURL
}

// findLinuxWheel returns the URL and SHA-256 digest of the CLI's Linux wheel for the
// version and architecture, read from the databricks project page of the index.
// https://peps.python.org/pep-0503/
func findLinuxWheel(ctx context.Context, indexURL *url.URL, architecture, version string) (*url.URL, string, error) {
	// Project pages live at <index>/<project>/, trailing slash included.
	pageURL := indexURL.JoinPath("databricks/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch the databricks package from %s: %w", indexURL.Host, errWithoutURL(err))
	}
	// Every index serves HTML; those that also serve JSON return HTML when asked for it.
	// https://peps.python.org/pep-0691/#content-types
	req.Header.Set("Accept", "text/html")
	resp, err := pypiHTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch the databricks package from %s: %w", indexURL.Host, errWithoutURL(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to fetch the databricks package from %s: HTTP %d", indexURL.Host, resp.StatusCode)
	}
	page, err := io.ReadAll(io.LimitReader(resp.Body, maxIndexPageSize+1))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read the databricks package page from %s: %w", indexURL.Host, err)
	}
	if len(page) > maxIndexPageSize {
		return nil, "", fmt.Errorf("the databricks package page from %s is larger than %d bytes", indexURL.Host, maxIndexPageSize)
	}
	hrefs, err := anchorHrefs(bytes.NewReader(page))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read the databricks package page from %s: %w", indexURL.Host, err)
	}

	machine := wheelMachines[architecture]
	for _, href := range hrefs {
		// Links are relative to the page, which redirects may have moved.
		wheelURL, err := resp.Request.URL.Parse(href)
		if err != nil || !isLinuxWheel(path.Base(wheelURL.Path), version, machine) {
			continue
		}
		digest, ok := strings.CutPrefix(wheelURL.Fragment, "sha256=")
		if !ok {
			return nil, "", fmt.Errorf("%s lists no SHA-256 digest for %s, so it cannot be verified", indexURL.Host, path.Base(wheelURL.Path))
		}
		// The index credentials belong to its host, as they do for pip: relative links
		// inherit them, and absolute links back to the same host get them here.
		if wheelURL.User == nil && wheelURL.Scheme == indexURL.Scheme && wheelURL.Host == indexURL.Host {
			wheelURL.User = indexURL.User
		}
		return wheelURL, digest, nil
	}
	return nil, "", fmt.Errorf("%s has no Linux %s wheel of databricks %s", indexURL.Host, machine, version)
}

// isLinuxWheel reports whether fileName is the CLI wheel for the version and Linux machine.
// Its platform part is a compressed tag set, such as
// manylinux_2_17_x86_64.manylinux2014_x86_64.musllinux_1_1_x86_64 (see tools/build_cli_wheels.py).
// https://packaging.python.org/en/latest/specifications/binary-distribution-format/#file-name-convention
func isLinuxWheel(fileName, version, machine string) bool {
	platforms, ok := strings.CutPrefix(fileName, "databricks-"+version+"-py3-none-")
	if !ok {
		return false
	}
	platforms, ok = strings.CutSuffix(platforms, ".whl")
	if !ok {
		return false
	}
	for tag := range strings.SplitSeq(platforms, ".") {
		if (strings.HasPrefix(tag, "manylinux") || strings.HasPrefix(tag, "musllinux")) && strings.HasSuffix(tag, "_"+machine) {
			return true
		}
	}
	return false
}

// anchorHrefs returns the href of every anchor on an index page.
func anchorHrefs(page io.Reader) ([]string, error) {
	var hrefs []string
	tokenizer := html.NewTokenizer(page)
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			if errors.Is(tokenizer.Err(), io.EOF) {
				return hrefs, nil
			}
			return nil, tokenizer.Err()
		}
		if tokenType != html.StartTagToken {
			continue
		}
		name, hasAttr := tokenizer.TagName()
		if string(name) != "a" {
			continue
		}
		for hasAttr {
			var key, value []byte
			key, value, hasAttr = tokenizer.TagAttr()
			if string(key) == "href" {
				hrefs = append(hrefs, string(value))
			}
		}
	}
}

// downloadWheel downloads the wheel and checks it against the SHA-256 digest the index lists.
func downloadWheel(ctx context.Context, wheelURL *url.URL, wantSHA256 string) ([]byte, error) {
	fileName := path.Base(wheelURL.Path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wheelURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s from %s: %w", fileName, wheelURL.Host, errWithoutURL(err))
	}
	log.Infof(ctx, "Downloading %s from %s", fileName, wheelURL.Host)
	resp, err := pypiHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s from %s: %w", fileName, wheelURL.Host, errWithoutURL(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download %s from %s: HTTP %d", fileName, wheelURL.Host, resp.StatusCode)
	}
	wheel, err := io.ReadAll(io.LimitReader(resp.Body, maxWheelSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to download %s from %s: %w", fileName, wheelURL.Host, err)
	}
	if len(wheel) > maxWheelSize {
		return nil, fmt.Errorf("%s from %s is larger than %d bytes", fileName, wheelURL.Host, maxWheelSize)
	}
	sum := sha256.Sum256(wheel)
	if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, wantSHA256) {
		return nil, fmt.Errorf("%s from %s has SHA-256 digest %s, but the index lists %s", fileName, wheelURL.Host, got, wantSHA256)
	}
	return wheel, nil
}

// releaseArchiveFromWheel returns a zip archive with the wheel's databricks binary at its
// root, the layout of the GitHub release archive that the workspace unpacks on upload. The
// binary is copied in its compressed form, as zip.Writer.Copy does, under its new name.
func releaseArchiveFromWheel(wheel []byte, version string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(wheel), int64(len(wheel)))
	if err != nil {
		return nil, fmt.Errorf("failed to read the databricks %s wheel: %w", version, err)
	}
	// The wheel installs the binary through its scripts directory (see tools/build_cli_wheels.py).
	binaryPath := "databricks-" + version + ".data/scripts/databricks"
	for _, file := range reader.File {
		if file.Name != binaryPath {
			continue
		}
		compressed, err := file.OpenRaw()
		if err != nil {
			return nil, fmt.Errorf("failed to read %s from the databricks %s wheel: %w", binaryPath, version, err)
		}
		header := file.FileHeader
		header.Name = "databricks"

		var archive bytes.Buffer
		writer := zip.NewWriter(&archive)
		entry, err := writer.CreateRaw(&header)
		if err != nil {
			return nil, fmt.Errorf("failed to build the release archive: %w", err)
		}
		if _, err := io.Copy(entry, compressed); err != nil {
			return nil, fmt.Errorf("failed to build the release archive: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("failed to build the release archive: %w", err)
		}
		return archive.Bytes(), nil
	}
	return nil, fmt.Errorf("the databricks %s wheel has no %s", version, binaryPath)
}

// errWithoutURL drops the request URL that net/http adds to its errors: the URL of a
// private index can carry credentials, which net/http masks only partially.
func errWithoutURL(err error) error {
	if uerr, ok := errors.AsType[*url.Error](err); ok {
		return uerr.Err
	}
	return err
}
