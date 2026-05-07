package lakebox

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	c    *client.DatabricksClient
	wsID string
}

// createRequest is the JSON body for POST /api/2.0/lakebox/sandboxes.
//
// The proto-defined `CreateSandboxRequest` carries a `Sandbox sandbox = 1`
// field today (every member is server-chosen), but JSON transcoding accepts
// the unwrapped form for forward-compatible callers. Keep `public_key` here
// as a no-op compat shim so older `lakebox create --public-key-file=...`
// invocations don't error — the manager ignores it on the wire.
type createRequest struct {
	PublicKey string `json:"public_key,omitempty"`
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
	SandboxID   string  `json:"sandboxId"`
	Status      string  `json:"status"`
	FQDN        string  `json:"fqdn"`
	IdleTimeout *string `json:"idleTimeout,omitempty"`
	NoAutostop  *bool   `json:"noAutostop,omitempty"`
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
type listResponse struct {
	Sandboxes []sandboxEntry `json:"sandboxes"`
}

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
	return &lakeboxAPI{c: c, wsID: resolveWorkspaceID(w)}, nil
}

// resolveWorkspaceID returns the workspace ID for the org-id header. Falls
// back to the `?o=<id>` query parameter on the Host because some staging
// gateways are configured that way and the SDK does not strip it into
// Config.WorkspaceID.
func resolveWorkspaceID(w *databricks.WorkspaceClient) string {
	if id := w.Config.WorkspaceID; id != "" {
		return id
	}
	parsed, err := url.Parse(w.Config.Host)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("o")
}

func (a *lakeboxAPI) headers() map[string]string {
	if a.wsID == "" {
		return nil
	}
	return map[string]string{orgIDHeader: a.wsID}
}

// create calls POST /api/2.0/lakebox/sandboxes with an optional public key.
func (a *lakeboxAPI) create(ctx context.Context, publicKey string) (*createResponse, error) {
	var resp createResponse
	err := a.c.Do(ctx, http.MethodPost, lakeboxAPIPath, a.headers(), nil, createRequest{PublicKey: publicKey}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// list calls GET /api/2.0/lakebox/sandboxes.
func (a *lakeboxAPI) list(ctx context.Context) ([]sandboxEntry, error) {
	var resp listResponse
	err := a.c.Do(ctx, http.MethodGet, lakeboxAPIPath, a.headers(), nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Sandboxes, nil
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
func (a *lakeboxAPI) update(ctx context.Context, id string, idleTimeoutSecs *int64, noAutostop *bool) (*sandboxEntry, error) {
	var idleTimeout *string
	if idleTimeoutSecs != nil {
		s := fmt.Sprintf("%ds", *idleTimeoutSecs)
		idleTimeout = &s
	}
	body := updateBody{
		SandboxID:   id,
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
