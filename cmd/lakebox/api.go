package lakebox

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

// Sandboxes live under the `/sandboxes` sub-collection of the lakebox service
// namespace (see `lakebox.proto` `LakeboxService.CreateSandbox`).
const lakeboxAPIPath = "/api/2.0/lakebox/sandboxes"

// SSH keys are nested under the lakebox service namespace alongside
// `sandboxes/` (see `LakeboxService.CreateSshKey`).
const lakeboxKeysAPIPath = "/api/2.0/lakebox/ssh-keys"

// orgIDHeader is sent by multi-workspace gateways (e.g. dogfood staging) so
// the gateway can scope the credential to a specific workspace. Without it,
// requests fail with "Credential was not sent or was of an unsupported type
// for this API."
const orgIDHeader = "X-Databricks-Org-Id"

// lakeboxAPI wraps the SDK ApiClient with workspace-id-aware request headers.
type lakeboxAPI struct {
	c *client.DatabricksClient
}

// sandboxCreateBody is the inner `Sandbox` message in the create payload.
// Only `name` is caller-settable today; all other fields are server-chosen.
type sandboxCreateBody struct {
	Name string `json:"name,omitempty"`
}

// createRequest is the JSON body for POST /api/2.0/lakebox/sandboxes.
// `CreateSandboxRequest { Sandbox sandbox = 1 }` has `body: "*"`, so the
// wire body is the full request with a `sandbox` wrapper.
type createRequest struct {
	Sandbox sandboxCreateBody `json:"sandbox"`
}

// createResponse is the JSON body returned by POST /api/2.0/lakebox/sandboxes.
// Mirrors the `Sandbox` proto message after JSON transcoding.
type createResponse struct {
	SandboxID string `json:"sandboxId"`
	Status    string `json:"status"`
	FQDN      string `json:"fqdn"`
}

// sandboxEntry is a single item in the list response.
// Mirrors the `Sandbox` proto message after JSON transcoding.
//
// IdleTimeout and NoAutostop correspond to the proto's `optional` fields;
// they're pointers so we can tell "field absent on the wire" (server has
// the global default) from "explicitly set to 0 / false."
//
// `IdleTimeout` is a `google.protobuf.Duration`. Proto3 JSON canonical
// form serializes Duration as a string with an `s` suffix (e.g.
// `"900s"`), so the Go field is `*string` and we parse on read.
type sandboxEntry struct {
	SandboxID     string  `json:"sandboxId"`
	Status        string  `json:"status"`
	FQDN          string  `json:"fqdn"`
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

// defaultAutoStopSecs mirrors the manager's `watchdog_idle_grace_secs`
// fallback (10 minutes) used when a sandbox has no per-record override.
// The value is also documented in `lakebox/CLAUDE.md` ("Sandbox
// Watchdog" section). Hardcoded here so list/status can render the
// effective timeout without an extra round-trip to fetch manager config.
const defaultAutoStopSecs int64 = 600

// autoStopLabel renders the auto-stop policy advertised by the manager
// for one sandbox into a short human-readable string. Mirrors the wire
// semantics from `lakebox/proto/lakebox.proto`:
//   - `no_autostop == true` → never auto-stops
//   - `idle_timeout` set and positive → that many seconds
//   - otherwise → manager's global default (`defaultAutoStopSecs`)
func (e *sandboxEntry) autoStopLabel() string {
	if e.NoAutostop != nil && *e.NoAutostop {
		return "never"
	}
	if secs := e.idleTimeoutSecs(); secs > 0 {
		return formatDurationSecs(secs)
	}
	return formatDurationSecs(defaultAutoStopSecs)
}

// formatDurationSecs prints `secs` as a compact duration (e.g. `90s`,
// `15m`, `2h`, `1h30m`). Falls back to seconds if it's not a clean
// minute/hour multiple. Avoids pulling in a dependency just for this.
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
// `nextPageToken` is empty on the final page (or when the result fits in one).
type listResponse struct {
	Sandboxes     []sandboxEntry `json:"sandboxes"`
	NextPageToken string         `json:"nextPageToken,omitempty"`
}

// listPageSize matches the manager-side default. Typical user fleets are
// well under this, so one round-trip covers them; the pagination loop in
// `list` handles the rare larger fleet.
const listPageSize = 100

// updateBody is the PATCH request body. The proto declares
// `UpdateSandboxRequest { Sandbox sandbox = 1 }` with `body: "sandbox"`
// in the (google.api.http) annotation, so the HTTP body is the inner
// `Sandbox` message directly — there is no `{"sandbox": {...}}`
// wrapping on the wire.
//
// Pointer fields encode the proto3 `optional` semantics — only the
// fields we explicitly set are emitted, leaving everything else
// server-untouched. `IdleTimeout` is a proto3-canonical Duration
// string (e.g. `"900s"`); the server-side wire type is
// `google.protobuf.Duration`.
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

func newLakeboxAPI(w *databricks.WorkspaceClient) (*lakeboxAPI, error) {
	c, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create lakebox API client: %w", err)
	}
	return &lakeboxAPI{c: c}, nil
}

// headers attaches the workspace routing identifier so multi-workspace
// gateways (e.g. SPOG hosts) can scope the credential. Mirrors the pattern
// in libs/telemetry, libs/filer, and SDK-generated workspace services. The
// auth.WorkspaceIDNone sentinel ("none") is treated as unset so the literal
// string never goes on the wire.
func (a *lakeboxAPI) headers() map[string]string {
	wsID := a.c.Config.WorkspaceID
	if wsID == "" || wsID == auth.WorkspaceIDNone {
		return nil
	}
	return map[string]string{orgIDHeader: wsID}
}

// create calls POST /api/2.0/lakebox/sandboxes. An empty `name` is omitted
// from the wire payload so the server treats it as "unset" rather than
// "explicit empty string."
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
// server stops sending `next_page_token`. Returns the full set in one slice.
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

// listPage fetches a single page of sandboxes. An empty `pageToken` requests
// the first page; the server enforces ordering across pages.
func (a *lakeboxAPI) listPage(ctx context.Context, pageToken string) (*listResponse, error) {
	query := map[string]any{"page_size": listPageSize}
	if pageToken != "" {
		query["page_token"] = pageToken
	}
	var resp listResponse
	err := a.c.Do(ctx, http.MethodGet, lakeboxAPIPath, a.headers(), query, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// get calls GET /api/2.0/lakebox/sandboxes/{id}.
func (a *lakeboxAPI) get(ctx context.Context, id string) (*sandboxEntry, error) {
	var resp sandboxEntry
	err := a.c.Do(ctx, http.MethodGet, lakeboxAPIPath+"/"+id, a.headers(), nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// update calls PATCH /api/2.0/lakebox/sandboxes/{id} with whichever of
// `idle_timeout` / `no_autostop` the caller chose to set. Fields left
// nil are omitted from the wire payload, so the server preserves their
// current values. Returns the refreshed `sandboxEntry`.
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
	err := a.c.Do(ctx, http.MethodPatch, lakeboxAPIPath+"/"+id, a.headers(), nil, body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// delete calls DELETE /api/2.0/lakebox/sandboxes/{id}.
func (a *lakeboxAPI) delete(ctx context.Context, id string) error {
	return a.c.Do(ctx, http.MethodDelete, lakeboxAPIPath+"/"+id, a.headers(), nil, nil, nil)
}

// registerKey calls POST /api/2.0/lakebox/ssh-keys.
func (a *lakeboxAPI) registerKey(ctx context.Context, publicKey string) error {
	return a.c.Do(ctx, http.MethodPost, lakeboxKeysAPIPath, a.headers(), nil, registerKeyRequest{PublicKey: publicKey}, nil)
}
