package lakebox

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

// sandboxPath returns the URL path for a single sandbox resource. The ID is
// path-escaped so a value like `foo;rm -rf /` lands on
// `/sandboxes/foo%3Brm%20-rf%20%2F` and gets a clean 400 from the server,
// rather than its unescaped `/` re-routing the request to the list endpoint
// (which silently returns an empty result the CLI then renders as an
// all-zero sandbox record).
func sandboxPath(id string) string {
	return lakeboxAPIPath + "/" + url.PathEscape(id)
}

// lakeboxAPIRoot is the service namespace under which the sandbox and
// ssh-key sub-collections live. Centralised so a server-side rename
// (e.g. "lakebox" → "sandbox") is a one-line change.
const lakeboxAPIRoot = "/api/2.0/lakebox"

// Sub-collections under the lakebox service namespace.
const (
	lakeboxAPIPath     = lakeboxAPIRoot + "/sandboxes"
	lakeboxKeysAPIPath = lakeboxAPIRoot + "/ssh-keys"
)

// orgIDHeader scopes the credential to a workspace on multi-workspace
// gateways. Without it, requests fail with "Credential was not sent or was
// of an unsupported type for this API."
const orgIDHeader = "X-Databricks-Org-Id"

// maxNameBytes mirrors the server-side `Sandbox.name` cap. The server
// measures bytes (not runes), so emoji hit the limit faster than expected;
// mirroring it client-side lets us fail fast with the observed byte count.
const maxNameBytes = 256

// validateName rejects names that exceed the wire limit (counted in bytes).
func validateName(name string) error {
	if n := len(name); n > maxNameBytes {
		return fmt.Errorf("--name is %d bytes; limit is %d (emoji and most non-ASCII characters count as 2-4 bytes each)", n, maxNameBytes)
	}
	return nil
}

// lakeboxAPI wraps the SDK ApiClient with workspace-id-aware request headers.
type lakeboxAPI struct {
	c *client.DatabricksClient
}

// sandboxCreateBody is the inner `Sandbox` message in the create payload.
// Only `name` is caller-settable; the rest are server-chosen.
type sandboxCreateBody struct {
	Name string `json:"name,omitempty"`
}

// createRequest is the wrapped POST body for sandbox creation.
type createRequest struct {
	Sandbox sandboxCreateBody `json:"sandbox"`
}

// createResponse mirrors the Sandbox proto after JSON transcoding.
// GatewayHost is `omitempty` so old and new server versions round-trip
// cleanly.
type createResponse struct {
	SandboxID   string `json:"sandboxId"`
	Status      string `json:"status"`
	GatewayHost string `json:"gatewayHost,omitempty"`
}

// sandboxEntry mirrors the Sandbox proto after JSON transcoding.
// IdleTimeout and NoAutostop are pointer-typed so we can distinguish
// "field absent on the wire" (server uses its default) from "explicitly
// set to 0 / false". IdleTimeout is a proto3-canonical Duration string
// (see idleTimeoutSecs).
type sandboxEntry struct {
	SandboxID     string  `json:"sandboxId"`
	Status        string  `json:"status"`
	GatewayHost   string  `json:"gatewayHost,omitempty"`
	Name          string  `json:"name,omitempty"`
	CreateTime    string  `json:"createTime,omitempty"`
	LastStartTime string  `json:"lastStartTime,omitempty"`
	IdleTimeout   *string `json:"idleTimeout,omitempty"`
	NoAutostop    *bool   `json:"noAutostop,omitempty"`
}

// idleTimeoutSecs parses the proto3-canonical Duration string off
// `IdleTimeout` (e.g. `"900s"` → `900`). Returns 0 when unset or when
// the string is not a recognizable Duration. Sub-second precision is
// dropped — the watchdog only acts on whole seconds.
func (e *sandboxEntry) idleTimeoutSecs() int64 {
	if e.IdleTimeout == nil {
		return 0
	}
	s := *e.IdleTimeout
	if !strings.HasSuffix(s, "s") {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return int64(d.Seconds())
}

// autoStopLabel renders the auto-stop policy for one sandbox:
//   - `no_autostop == true` → never auto-stops
//   - `idle_timeout` set and positive → that many seconds
//   - otherwise → no enforcement today; render as "never"
//
// If the manager later enforces an idle-grace default, render it here.
func (e *sandboxEntry) autoStopLabel() string {
	if e.NoAutostop != nil && *e.NoAutostop {
		return "never"
	}
	if secs := e.idleTimeoutSecs(); secs > 0 {
		return formatDurationSecs(secs)
	}
	return "never"
}

// formatDurationSecs prints `secs` as a compact duration (e.g. `90s`,
// `15m`, `2h`, `1h30m`). Falls back to seconds if it's not a clean
// minute/hour multiple.
func formatDurationSecs(secs int64) string {
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	if secs%3600 == 0 {
		return fmt.Sprintf("%dh", secs/3600)
	}
	if secs >= 3600 {
		return fmt.Sprintf("%dh%dm", secs/3600, (secs%3600)/60)
	}
	if secs%60 == 0 {
		return fmt.Sprintf("%dm", secs/60)
	}
	return fmt.Sprintf("%ds", secs)
}

// listResponse is the JSON body returned by GET /api/2.0/lakebox/sandboxes.
type listResponse struct {
	Sandboxes     []sandboxEntry `json:"sandboxes"`
	NextPageToken string         `json:"nextPageToken,omitempty"`
}

// listPageSize matches the manager-side default.
const listPageSize = 100

// updateBody is the PATCH body; the server takes the inner `Sandbox`
// message directly with no `{"sandbox": ...}` wrapping. Pointer fields
// encode proto3 optional semantics (see sandboxEntry).
type updateBody struct {
	SandboxID   string  `json:"sandbox_id"`
	Name        *string `json:"name,omitempty"`
	IdleTimeout *string `json:"idle_timeout,omitempty"`
	NoAutostop  *bool   `json:"no_autostop,omitempty"`
}

// registerKeyRequest is the JSON body for POST /api/2.0/lakebox/ssh-keys.
type registerKeyRequest struct {
	PublicKey string `json:"public_key"`
	Name      string `json:"name,omitempty"`
}

// newLakeboxAPI returns a lakeboxAPI bound to the workspace client's config.
func newLakeboxAPI(w *databricks.WorkspaceClient) (*lakeboxAPI, error) {
	c, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create lakebox API client: %w", err)
	}
	return &lakeboxAPI{c: c}, nil
}

// headers attaches the workspace routing identifier so multi-workspace
// gateways (e.g. SPOG hosts) can scope the credential. The
// auth.WorkspaceIDNone sentinel ("none") is treated as unset so the
// literal string never goes on the wire.
func (a *lakeboxAPI) headers() map[string]string {
	wsID := a.c.Config.WorkspaceID
	if wsID == "" || wsID == auth.WorkspaceIDNone {
		return nil
	}
	return map[string]string{orgIDHeader: wsID}
}

// create calls POST /api/2.0/lakebox/sandboxes. An empty `name` is omitted
// so the server treats it as "unset" rather than "explicit empty string".
func (a *lakeboxAPI) create(ctx context.Context, name string) (*createResponse, error) {
	body := createRequest{Sandbox: sandboxCreateBody{Name: name}}
	var resp createResponse
	err := a.c.Do(ctx, http.MethodPost, lakeboxAPIPath, a.headers(), nil, body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// list calls GET /api/2.0/lakebox/sandboxes, following pagination until the
// server stops sending `next_page_token`.
func (a *lakeboxAPI) list(ctx context.Context) ([]sandboxEntry, error) {
	var all []sandboxEntry
	pageToken := ""
	for {
		page, err := a.listPage(ctx, pageToken)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Sandboxes...)
		if page.NextPageToken == "" {
			return all, nil
		}
		pageToken = page.NextPageToken
	}
}

// listPage fetches a single page of sandboxes.
//
// `query` is passed in slot 6 (`request`), not slot 5 (`queryParams`). On
// GET, the SDK's makeRequestBody serializes `request` into the URL query
// string and sends an empty body. Routing through `queryParams` instead
// makes it write a literal `null` body, which the lakebox manager rejects
// with `INVALID_PARAMETER_VALUE: Request body must be a JSON object`. See
// databricks-sdk-go/httpclient/request.go:makeRequestBody.
func (a *lakeboxAPI) listPage(ctx context.Context, pageToken string) (*listResponse, error) {
	query := map[string]any{"page_size": listPageSize}
	if pageToken != "" {
		query["page_token"] = pageToken
	}
	var resp listResponse
	err := a.c.Do(ctx, http.MethodGet, lakeboxAPIPath, a.headers(), nil, query, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// get calls GET /api/2.0/lakebox/sandboxes/{id}.
func (a *lakeboxAPI) get(ctx context.Context, id string) (*sandboxEntry, error) {
	var resp sandboxEntry
	err := a.c.Do(ctx, http.MethodGet, sandboxPath(id), a.headers(), nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// update calls PATCH /api/2.0/lakebox/sandboxes/{id} with whichever of
// `idle_timeout` / `no_autostop` the caller chose to set. Fields left nil
// are omitted from the wire payload, so the server preserves their current
// values. Returns the refreshed `sandboxEntry`.
func (a *lakeboxAPI) update(ctx context.Context, id string, name *string, idleTimeoutSecs *int64, noAutostop *bool) (*sandboxEntry, error) {
	var idleTimeout *string
	if idleTimeoutSecs != nil {
		s := fmt.Sprintf("%ds", *idleTimeoutSecs)
		idleTimeout = &s
	}
	body := updateBody{
		SandboxID:   id,
		Name:        name,
		IdleTimeout: idleTimeout,
		NoAutostop:  noAutostop,
	}
	var resp sandboxEntry
	err := a.c.Do(ctx, http.MethodPatch, sandboxPath(id), a.headers(), nil, body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// delete calls DELETE /api/2.0/lakebox/sandboxes/{id}.
func (a *lakeboxAPI) delete(ctx context.Context, id string) error {
	return a.c.Do(ctx, http.MethodDelete, sandboxPath(id), a.headers(), nil, nil, nil)
}

// stop calls POST /api/2.0/lakebox/sandboxes/{id}/stop and returns the
// refreshed sandbox.
func (a *lakeboxAPI) stop(ctx context.Context, id string) (*sandboxEntry, error) {
	body := map[string]string{"sandbox_id": id}
	var resp sandboxEntry
	err := a.c.Do(ctx, http.MethodPost, sandboxPath(id)+"/stop", a.headers(), nil, body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// start calls POST /api/2.0/lakebox/sandboxes/{id}/start and returns the
// refreshed sandbox.
func (a *lakeboxAPI) start(ctx context.Context, id string) (*sandboxEntry, error) {
	body := map[string]string{"sandbox_id": id}
	var resp sandboxEntry
	err := a.c.Do(ctx, http.MethodPost, sandboxPath(id)+"/start", a.headers(), nil, body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// registerKey calls POST /api/2.0/lakebox/ssh-keys. An empty `name` is
// omitted so the server records "unset" rather than an explicit empty string.
func (a *lakeboxAPI) registerKey(ctx context.Context, publicKey, name string) error {
	return a.c.Do(ctx, http.MethodPost, lakeboxKeysAPIPath, a.headers(), nil, registerKeyRequest{PublicKey: publicKey, Name: name}, nil)
}

// sshKeyEntry is a single item in the ssh-key list response.
type sshKeyEntry struct {
	KeyHash     string `json:"keyHash"`
	Name        string `json:"name,omitempty"`
	CreateTime  string `json:"createTime,omitempty"`
	LastUseTime string `json:"lastUseTime,omitempty"`
}

// listKeysResponse is the JSON body returned by GET /api/2.0/lakebox/ssh-keys.
// Per-user keys are hard-capped server-side, so the full set fits in one
// response — no pagination.
type listKeysResponse struct {
	SshKeys []sshKeyEntry `json:"sshKeys"`
}

// listKeys calls GET /api/2.0/lakebox/ssh-keys.
func (a *lakeboxAPI) listKeys(ctx context.Context) ([]sshKeyEntry, error) {
	var resp listKeysResponse
	err := a.c.Do(ctx, http.MethodGet, lakeboxKeysAPIPath, a.headers(), nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.SshKeys, nil
}

// deleteKey calls DELETE /api/2.0/lakebox/ssh-keys/{key_hash}.
func (a *lakeboxAPI) deleteKey(ctx context.Context, keyHash string) error {
	return a.c.Do(ctx, http.MethodDelete, lakeboxKeysAPIPath+"/"+url.PathEscape(keyHash), a.headers(), nil, nil, nil)
}
