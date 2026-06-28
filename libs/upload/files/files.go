// Package files is the authenticated Databricks Files API client used by the
// large-file uploader. It owns everything that talks to /api/2.0/fs.
//
// It is implemented as a wrapper on top of the SDK Files client to add support
// for methods that have not been added to the SDK yet.
//
// TODO: This client should ultimately be entirely replaced by the SDK Files
// client once it has reached feature parity.
package files

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	sdkapierr "github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/httpclient"
	sdkfiles "github.com/databricks/databricks-sdk-go/service/files"
	"github.com/databricks/sdk-go/core/apierr"
	"github.com/databricks/sdk-go/core/apierr/codes"
	"github.com/databricks/sdk-go/core/apiretry"
	"github.com/databricks/sdk-go/core/ops"

	"github.com/databricks/databricks-sdk-go/config/credentials"
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

// controlPlaneBackoff is the exponential-backoff-with-jitter policy for
// control-plane retries.
var controlPlaneBackoff = ops.BackoffPolicy{}

// controlPlaneRetryTimeout bounds the total time spent retrying a single
// control-plane call. Without it, a persistently failing endpoint would be
// retried until the caller's context expires (and forever if it has no
// deadline).
var controlPlaneRetryTimeout = 5 * time.Minute

// FilesClient is the subset of the SDK Files API used for the standard
// single-shot upload.
//
// A *databricks.WorkspaceClient's Files field satisfies it directly,
// while staying an in-package interface this package owns and tests
// can fake.
type FilesClient interface {
	Upload(ctx context.Context, request sdkfiles.UploadRequest) error
}

// Client coordinates large uploads with the Databricks Files API. The standard
// single-shot upload goes through the SDK FilesClient; the large-file multipart
// and resumable coordination runs over a custom authenticated HTTP client, and
// the parts/chunks transfer directly to cloud storage (in the cloudstorage package)
// over the URLs this client mints.
type Client struct {
	files FilesClient
	host  string
	creds credentials.CredentialsProvider
	http  *http.Client
}

// ClientOptions configures a [Client].
type ClientOptions struct {
	// HTTPClient is the HTTP client to use for control-plane requests.
	HTTPClient *http.Client

	// CredentialsProvider is the credentials provider to use for control-plane
	// requests.
	//
	// IMPORTANT: that provider must apply the workspace routing headers to the
	// request by setting the X-Databricks-Workspace-Id header.
	CredentialsProvider credentials.CredentialsProvider

	// Host is the host to use for control-plane requests.
	Host string

	// FilesClient is the FilesClient to use for single-shot uploads.
	FilesClient FilesClient
}

// New returns a Client from options. options.FilesClient performs the standard
// single-shot upload; options.Host and options.CredentialsProvider address and
// authenticate the custom multipart/resumable control-plane requests (a
// WorkspaceClient's Config.Authenticate satisfies the provider via
// credentials.CredentialsProviderFn). options.HTTPClient carries the
// control-plane transport and defaults to http.DefaultClient; the cloud part
// transfers are unauthenticated and never see the credentials.
func New(options ClientOptions) (*Client, error) {
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}
	if options.FilesClient == nil {
		return nil, errors.New("SDK Files client must not be nil")
	}
	if options.Host == "" {
		return nil, errors.New("host must not be empty")
	}
	if options.CredentialsProvider == nil {
		return nil, errors.New("credentials provider must not be nil")
	}
	return &Client{
		files: options.FilesClient,
		host:  options.Host,
		creds: options.CredentialsProvider,
		http:  options.HTTPClient,
	}, nil
}

// Upload sends the whole body in one PUT through the SDK Files client. Used for
// small files and as the multipart/resumable fallback. overwrite is tri-state:
// a non-nil value is sent explicitly; nil lets the server apply its default. A
// 409 ALREADY_EXISTS is mapped to ErrAlreadyExists.
func (c *Client) Upload(ctx context.Context, path string, overwrite *bool, body io.Reader) error {
	req := sdkfiles.UploadRequest{
		FilePath: path,
		Contents: io.NopCloser(body),
	}
	// ForceSendFields makes the SDK send overwrite=false explicitly; omitting
	// both lets the server apply its default.
	if overwrite != nil {
		req.Overwrite = *overwrite
		req.ForceSendFields = []string{"Overwrite"}
	}
	return asAlreadyExists(c.files.Upload(ctx, req))
}

// Initiate opens a server-coordinated upload session for path. The result
// reports which protocol the workspace selected (multipart on AWS/Azure,
// resumable on GCP).
func (c *Client) Initiate(ctx context.Context, path string, overwrite *bool) (*InitiateResult, error) {
	query := url.Values{"action": {"initiate-upload"}}
	if overwrite != nil {
		query.Set("overwrite", strconv.FormatBool(*overwrite))
	}
	var out InitiateResult
	if err := c.controlPlaneJSON(ctx, http.MethodPost, filesAPIPath(path), query, nil, &out); err != nil {
		return nil, asAlreadyExists(err)
	}
	return &out, nil
}

// asAlreadyExists maps an "already exists" API error to the ErrAlreadyExists
// sentinel, so callers detect it with errors.Is regardless of which path
// surfaced it. The Files API returns HTTP 409 with error_code ALREADY_EXISTS
// when an upload targets an existing path with overwrite=false; for multipart
// this surfaces from complete-upload (after the parts are sent).
//
// Two error types are checked: the control plane runs over core/apierr, while
// single-shot Upload goes through the SDK Files client, whose older
// databricks-sdk-go apierr type core/apierr.Code does not classify. Other errors
// pass through unchanged.
func asAlreadyExists(err error) error {
	if apierr.Code(err) == codes.AlreadyExists {
		return ErrAlreadyExists
	}
	if aerr, ok := errors.AsType[*sdkapierr.APIError](err); ok &&
		aerr.StatusCode == http.StatusConflict && aerr.ErrorCode == "ALREADY_EXISTS" {
		return ErrAlreadyExists
	}
	return err
}

// CreatePartURLs mints count presigned URLs for multipart parts starting at
// startPart.
func (c *Client) CreatePartURLs(ctx context.Context, path, token string, startPart, count int) ([]PresignedURL, error) {
	body := createPartURLsRequest{
		Path:            path,
		SessionToken:    token,
		StartPartNumber: startPart,
		Count:           count,
		ExpireTime:      expireTime(),
	}
	var out createPartURLsResponse
	if err := c.controlPlaneJSON(ctx, http.MethodPost, "/api/2.0/fs/create-upload-part-urls", nil, body, &out); err != nil {
		return nil, err
	}
	if len(out.UploadPartURLs) == 0 {
		return nil, fmt.Errorf("%w: no upload part URLs returned", ErrUnexpectedServerResponse)
	}
	result := make([]PresignedURL, 0, len(out.UploadPartURLs))
	for _, p := range out.UploadPartURLs {
		result = append(result, PresignedURL{URL: p.URL, PartNumber: p.PartNumber, Headers: headerMap(p.Headers)})
	}
	return result, nil
}

// CreateResumableURL mints the single presigned URL for a GCP resumable session.
func (c *Client) CreateResumableURL(ctx context.Context, path, token string) (PresignedURL, error) {
	body := resumableURLRequest{Path: path, SessionToken: token}
	var out resumableURLResponse
	if err := c.controlPlaneJSON(ctx, http.MethodPost, "/api/2.0/fs/create-resumable-upload-url", nil, body, &out); err != nil {
		return PresignedURL{}, err
	}
	if out.ResumableUploadURL == nil || out.ResumableUploadURL.URL == "" {
		return PresignedURL{}, fmt.Errorf("%w: no resumable upload URL returned", ErrUnexpectedServerResponse)
	}
	return PresignedURL{URL: out.ResumableUploadURL.URL, Headers: headerMap(out.ResumableUploadURL.Headers)}, nil
}

// CompleteMultipart finalizes a multipart upload with the part ETags, which it
// sorts by part number before sending.
func (c *Client) CompleteMultipart(ctx context.Context, path, token string, etags map[int]string) error {
	nums := make([]int, 0, len(etags))
	for n := range etags {
		nums = append(nums, n)
	}
	slices.Sort(nums)
	parts := make([]completePart, 0, len(nums))
	for _, n := range nums {
		parts = append(parts, completePart{PartNumber: n, ETag: etags[n]})
	}
	query := url.Values{"action": {"complete-upload"}, "upload_type": {"multipart"}, "session_token": {token}}
	return asAlreadyExists(c.controlPlaneJSON(ctx, http.MethodPost, filesAPIPath(path), query, completeRequest{Parts: parts}, nil))
}

// CreateAbortURL mints a presigned URL for aborting an upload session. The
// caller issues the (unauthenticated) cloud DELETE against it.
func (c *Client) CreateAbortURL(ctx context.Context, path, token string) (PresignedURL, error) {
	body := abortURLRequest{Path: path, SessionToken: token, ExpireTime: expireTime()}
	var out abortURLResponse
	if err := c.controlPlaneJSON(ctx, http.MethodPost, "/api/2.0/fs/create-abort-upload-url", nil, body, &out); err != nil {
		return PresignedURL{}, err
	}
	if out.AbortUploadURL == nil || out.AbortUploadURL.URL == "" {
		return PresignedURL{}, fmt.Errorf("%w: no abort upload URL returned", ErrUnexpectedServerResponse)
	}
	return PresignedURL{URL: out.AbortUploadURL.URL, Headers: headerMap(out.AbortUploadURL.Headers)}, nil
}

// controlPlaneJSON performs an authenticated JSON request against the Files API
// control plane, retrying transient failures via core/ops. reqBody and out may
// be nil. A fresh request is built on each attempt so retries re-apply
// credentials (handling token refresh) and rewind the body.
func (c *Client) controlPlaneJSON(ctx context.Context, method, path string, query url.Values, reqBody, out any) error {
	urlStr := strings.TrimRight(c.host, "/") + path
	if len(query) > 0 {
		urlStr += "?" + query.Encode()
	}
	var bodyBytes []byte
	if reqBody != nil {
		var err error
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
		if err := c.creds.SetHeaders(req); err != nil {
			return err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if apiErr := apierr.FromHTTPError(resp.StatusCode, resp.Header, respBody); apiErr != nil {
			return apiErr
		}
		if out != nil && len(respBody) > 0 {
			return json.Unmarshal(respBody, out)
		}
		return nil
	}
	return ops.Execute(ctx, call, ops.WithRetrier(newControlPlaneRetrier), ops.WithTimeout(controlPlaneRetryTimeout))
}

func newControlPlaneRetrier() ops.Retrier {
	return apiretry.NewRetrier(controlPlaneBackoff, apiretry.RetrierConfig{StatusCodes: controlPlaneRetryStatusCodes})
}

// nowFunc defaults to real time; overridable in tests for deterministic
// presigned-URL expiry timestamps.
var nowFunc = time.Now

// uploadURLExpiry is how long requested presigned URLs are valid.
var uploadURLExpiry = time.Hour

// filesAPIPath builds the Files API URL path for an absolute volume path. It
// uses the SDK's segment encoder so the control-plane initiate/complete calls
// escape the path identically to the single-shot FilesClient.Upload.
func filesAPIPath(absPath string) string {
	return "/api/2.0/fs/files" + httpclient.EncodeMultiSegmentPathParameter(absPath)
}

func expireTime() string {
	return nowFunc().UTC().Add(uploadURLExpiry).Format("2006-01-02T15:04:05Z")
}

func headerMap(headers []nameValue) map[string]string {
	out := make(map[string]string, len(headers))
	for _, h := range headers {
		out[h.Name] = h.Value
	}
	return out
}
