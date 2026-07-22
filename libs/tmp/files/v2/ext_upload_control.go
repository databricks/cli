package files

// Files API control-plane calls for large uploads: initiate a server-coordinated
// session, mint presigned URLs for parts (multipart) and the resumable session
// (GCP), complete a multipart upload, and mint abort URLs for cleanup. It also
// owns the single-shot octet-stream PUT used for small files and as the
// multipart/resumable fallback.
//
// These calls run over the client's own authenticated transport (the same one
// the generated methods use); the parts/chunks then transfer directly to cloud
// storage over the URLs minted here (see the cloudstorage subpackage), carrying
// no Databricks credentials.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/databricks/sdk-go/core/apierr"
	"github.com/databricks/sdk-go/core/apierr/codes"
	"github.com/databricks/sdk-go/core/apiretry"
	"github.com/databricks/sdk-go/core/ops"
)

// Sentinel errors surfaced by the large-file upload API.
var (
	// errUnexpectedServerResponse is returned when the Files API control plane
	// returns a well-formed but semantically incomplete response (for example,
	// no presigned URLs). It is an internal signal, not a caller-facing sentinel.
	errUnexpectedServerResponse = errors.New("unexpected server response")

	// ErrAlreadyExists is returned when an upload targets an existing path with
	// overwrite disabled. It is surfaced as a sentinel so callers detect
	// "already exists" with errors.Is regardless of which protocol produced it.
	ErrAlreadyExists = errors.New("the file being created already exists")
)

// controlPlaneRetryStatusCodes are the HTTP statuses retried for an
// authenticated control-plane call.
var controlPlaneRetryStatusCodes = []int{
	http.StatusRequestTimeout,     // 408
	http.StatusTooManyRequests,    // 429
	http.StatusBadGateway,         // 502
	http.StatusServiceUnavailable, // 503
	http.StatusGatewayTimeout,     // 504
}

// --- Result types ---

// initiateResult reports which large-file protocol the server selected. Exactly
// one of multipartUpload (AWS/Azure) and resumableUpload (GCP) is non-nil on a
// well-formed response; the pointers distinguish "not offered" from "offered
// with an empty token".
type initiateResult struct {
	MultipartUpload *uploadSession `json:"multipart_upload"`
	ResumableUpload *uploadSession `json:"resumable_upload"`
}

type uploadSession struct {
	SessionToken string `json:"session_token"`
}

// presignedURL is a resolved cloud-storage URL with its associated request
// headers, as minted by the control plane and transferred over by the caller.
type presignedURL struct {
	URL        string
	PartNumber int
	Headers    map[string]string
}

// --- Wire types for the Files API multipart/resumable coordination ---

type nameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type createPartURLsRequest struct {
	Path            string `json:"path"`
	SessionToken    string `json:"session_token"`
	StartPartNumber int    `json:"start_part_number"`
	Count           int    `json:"count"`
	ExpireTime      string `json:"expire_time"`
}

type createPartURLsResponse struct {
	UploadPartURLs []struct {
		URL        string      `json:"url"`
		PartNumber int         `json:"part_number"`
		Headers    []nameValue `json:"headers"`
	} `json:"upload_part_urls"`
}

type resumableURLRequest struct {
	Path         string `json:"path"`
	SessionToken string `json:"session_token"`
}

// urlWithHeaders is a presigned cloud-storage URL and its required request
// headers, as returned by the resumable and abort URL endpoints.
type urlWithHeaders struct {
	URL     string      `json:"url"`
	Headers []nameValue `json:"headers"`
}

type resumableURLResponse struct {
	ResumableUploadURL *urlWithHeaders `json:"resumable_upload_url"`
}

type abortURLRequest struct {
	Path         string `json:"path"`
	SessionToken string `json:"session_token"`
	ExpireTime   string `json:"expire_time"`
}

type abortURLResponse struct {
	AbortUploadURL *urlWithHeaders `json:"abort_upload_url"`
}

type completePart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

type completeRequest struct {
	Parts []completePart `json:"parts"`
}

// uploadSingleShot sends the whole body in one PUT. Used for small files and as
// the multipart/resumable fallback. overwrite is tri-state: a non-nil value is
// sent explicitly; nil lets the server apply its default. A 409 ALREADY_EXISTS
// is mapped to ErrAlreadyExists.
//
// The generated UploadFile is unsuitable here: it JSON-marshals the reader
// rather than streaming the raw octet stream the Files API expects. A seekable
// body is rewound before each retry; a non-seekable body cannot be replayed, so
// it is attempted once.
func (e *engine) uploadSingleShot(ctx context.Context, path string, overwrite *bool, body io.Reader) error {
	query := url.Values{}
	if overwrite != nil {
		query.Set("overwrite", strconv.FormatBool(*overwrite))
	}
	urlStr, err := e.controlPlaneURL(filesAPIPath(path), query)
	if err != nil {
		return err
	}

	seeker, seekable := body.(io.Seeker)
	var start int64
	if seekable {
		if start, err = seeker.Seek(0, io.SeekCurrent); err != nil {
			seekable = false
		}
	}

	call := func(ctx context.Context) error {
		if seekable {
			if _, err := seeker.Seek(start, io.SeekStart); err != nil {
				return fmt.Errorf("rewinding upload body: %w", err)
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, urlStr, body)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		e.setWorkspaceHeader(req)
		_, _, err = executeHTTPCall(httpCallOptions{req: req, client: e.c.httpClient, logger: e.c.logger})
		return err
	}

	// Only retry when the body can be rewound; a consumed non-seekable stream
	// cannot be replayed, so it is attempted exactly once.
	opts := []ops.Option{ops.WithTimeout(e.tun.cpRetryTimeout)}
	if seekable {
		opts = append(opts, ops.WithRetrier(e.newControlPlaneRetrier))
	}
	return asAlreadyExists(ops.Execute(ctx, call, opts...))
}

// initiateUpload opens a server-coordinated upload session for path. The result
// reports which protocol the workspace selected (multipart on AWS/Azure,
// resumable on GCP).
func (e *engine) initiateUpload(ctx context.Context, path string, overwrite *bool) (*initiateResult, error) {
	query := url.Values{"action": {"initiate-upload"}}
	if overwrite != nil {
		query.Set("overwrite", strconv.FormatBool(*overwrite))
	}
	var out initiateResult
	if err := e.controlPlaneJSON(ctx, http.MethodPost, filesAPIPath(path), query, nil, &out); err != nil {
		return nil, asAlreadyExists(err)
	}
	return &out, nil
}

// createPartURLs mints count presigned URLs for multipart parts starting at
// startPart.
func (e *engine) createPartURLs(ctx context.Context, path, token string, startPart, count int) ([]presignedURL, error) {
	body := createPartURLsRequest{
		Path:            path,
		SessionToken:    token,
		StartPartNumber: startPart,
		Count:           count,
		ExpireTime:      e.expireTime(),
	}
	var out createPartURLsResponse
	if err := e.controlPlaneJSON(ctx, http.MethodPost, "/api/2.0/fs/create-upload-part-urls", nil, body, &out); err != nil {
		return nil, err
	}
	if len(out.UploadPartURLs) == 0 {
		return nil, fmt.Errorf("%w: no upload part URLs returned", errUnexpectedServerResponse)
	}
	result := make([]presignedURL, 0, len(out.UploadPartURLs))
	for _, p := range out.UploadPartURLs {
		result = append(result, presignedURL{URL: p.URL, PartNumber: p.PartNumber, Headers: headerMap(p.Headers)})
	}
	return result, nil
}

// createResumableURL mints the single presigned URL for a GCP resumable session.
func (e *engine) createResumableURL(ctx context.Context, path, token string) (presignedURL, error) {
	body := resumableURLRequest{Path: path, SessionToken: token}
	var out resumableURLResponse
	if err := e.controlPlaneJSON(ctx, http.MethodPost, "/api/2.0/fs/create-resumable-upload-url", nil, body, &out); err != nil {
		return presignedURL{}, err
	}
	if out.ResumableUploadURL == nil || out.ResumableUploadURL.URL == "" {
		return presignedURL{}, fmt.Errorf("%w: no resumable upload URL returned", errUnexpectedServerResponse)
	}
	return presignedURL{URL: out.ResumableUploadURL.URL, Headers: headerMap(out.ResumableUploadURL.Headers)}, nil
}

// completeMultipart finalizes a multipart upload with the part ETags, which it
// sorts by part number before sending.
func (e *engine) completeMultipart(ctx context.Context, path, token string, etags map[int]string) error {
	nums := slices.Sorted(maps.Keys(etags))
	parts := make([]completePart, 0, len(nums))
	for _, n := range nums {
		parts = append(parts, completePart{PartNumber: n, ETag: etags[n]})
	}
	query := url.Values{"action": {"complete-upload"}, "upload_type": {"multipart"}, "session_token": {token}}
	return asAlreadyExists(e.controlPlaneJSON(ctx, http.MethodPost, filesAPIPath(path), query, completeRequest{Parts: parts}, nil))
}

// createAbortURL mints a presigned URL for aborting an upload session. The
// caller issues the (unauthenticated) cloud DELETE against it.
func (e *engine) createAbortURL(ctx context.Context, path, token string) (presignedURL, error) {
	body := abortURLRequest{Path: path, SessionToken: token, ExpireTime: e.expireTime()}
	var out abortURLResponse
	if err := e.controlPlaneJSON(ctx, http.MethodPost, "/api/2.0/fs/create-abort-upload-url", nil, body, &out); err != nil {
		return presignedURL{}, err
	}
	if out.AbortUploadURL == nil || out.AbortUploadURL.URL == "" {
		return presignedURL{}, fmt.Errorf("%w: no abort upload URL returned", errUnexpectedServerResponse)
	}
	return presignedURL{URL: out.AbortUploadURL.URL, Headers: headerMap(out.AbortUploadURL.Headers)}, nil
}

// controlPlaneJSON performs an authenticated JSON request against the Files API
// control plane, retrying transient failures via core/ops. reqBody and out may
// be nil. A fresh request is built on each attempt so retries re-apply
// credentials (the auth transport handles token refresh) and rewind the body.
func (e *engine) controlPlaneJSON(ctx context.Context, method, path string, query url.Values, reqBody, out any) error {
	urlStr, err := e.controlPlaneURL(path, query)
	if err != nil {
		return err
	}
	var bodyBytes []byte
	if reqBody != nil {
		if bodyBytes, err = json.Marshal(reqBody); err != nil {
			return err
		}
	}

	call := func(ctx context.Context) error {
		var rdr io.Reader
		if bodyBytes != nil {
			rdr = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, urlStr, rdr)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		e.setWorkspaceHeader(req)
		respBody, _, err := executeHTTPCall(httpCallOptions{req: req, client: e.c.httpClient, logger: e.c.logger})
		if err != nil {
			return err
		}
		if out != nil && len(respBody) > 0 {
			return json.Unmarshal(respBody, out)
		}
		return nil
	}
	return ops.Execute(ctx, call, ops.WithRetrier(e.newControlPlaneRetrier), ops.WithTimeout(e.tun.cpRetryTimeout))
}

func (e *engine) newControlPlaneRetrier() ops.Retrier {
	return apiretry.NewRetrier(e.tun.cpBackoff, apiretry.RetrierConfig{StatusCodes: controlPlaneRetryStatusCodes})
}

// setWorkspaceHeader applies the workspace routing header when the client is
// workspace-scoped. The Files API control plane routes a request to the right
// workspace by this header; the auth transport supplies only the credentials.
func (e *engine) setWorkspaceHeader(req *http.Request) {
	if e.c.workspaceID != "" {
		req.Header.Set("X-Databricks-Workspace-Id", e.c.workspaceID)
	}
}

// controlPlaneURL joins the client host with a path and optional query,
// mirroring the generated client's URL construction.
func (e *engine) controlPlaneURL(path string, query url.Values) (string, error) {
	baseURL, err := url.Parse(e.c.host)
	if err != nil {
		return "", err
	}
	baseURL.Path = path
	if len(query) > 0 {
		baseURL.RawQuery = query.Encode()
	}
	return baseURL.String(), nil
}

// asAlreadyExists maps an "already exists" API error to the ErrAlreadyExists
// sentinel, so callers detect it with errors.Is regardless of which path
// surfaced it. The Files API returns HTTP 409 with error_code ALREADY_EXISTS
// when an upload targets an existing path with overwrite=false; for multipart
// this surfaces from complete-upload (after the parts are sent). Every path runs
// over core/apierr, so a single code check suffices. Other errors pass through
// unchanged.
func asAlreadyExists(err error) error {
	if apierr.Code(err) == codes.AlreadyExists {
		return ErrAlreadyExists
	}
	return err
}

// filesAPIPath builds the Files API URL path for an absolute volume path. It
// matches the single-shot UploadFile path construction so the control-plane
// initiate/complete calls escape the path identically.
func filesAPIPath(absPath string) string {
	return fmt.Sprintf("/api/2.0/fs/files%v", absPath)
}

// expireTime is the presigned-URL expiry timestamp sent to the control plane,
// urlExpiry into the future.
func (e *engine) expireTime() string {
	return time.Now().UTC().Add(e.tun.urlExpiry).Format("2006-01-02T15:04:05Z")
}

func headerMap(headers []nameValue) map[string]string {
	out := make(map[string]string, len(headers))
	for _, h := range headers {
		out[h.Name] = h.Value
	}
	return out
}
