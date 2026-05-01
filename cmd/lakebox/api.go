package lakebox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go"
)

// Sandboxes live under the `/sandboxes` sub-collection of the lakebox service
// namespace (see `lakebox.proto` `LakeboxService.CreateSandbox`).
const lakeboxAPIPath = "/api/2.0/lakebox/sandboxes"

// lakeboxAPI wraps raw HTTP calls to the lakebox REST API.
type lakeboxAPI struct {
	w *databricks.WorkspaceClient
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
// IdleTimeoutSecs and Persist correspond to the proto's `optional` fields;
// they're pointers so we can tell "field absent on the wire" (server has the
// global default) from "explicitly set to 0 / false."
type sandboxEntry struct {
	SandboxID       string `json:"sandboxId"`
	Status          string `json:"status"`
	FQDN            string `json:"fqdn"`
	IdleTimeoutSecs *int64 `json:"idleTimeoutSecs,omitempty"`
	Persist         *bool  `json:"persist,omitempty"`
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
//   - `persist == true` → never auto-stops
//   - `idle_timeout_secs` set and positive → that many seconds
//   - otherwise → manager's global default (`defaultAutoStopSecs`)
func (e *sandboxEntry) autoStopLabel() string {
	if e.Persist != nil && *e.Persist {
		return "never"
	}
	if e.IdleTimeoutSecs != nil && *e.IdleTimeoutSecs > 0 {
		return formatDurationSecs(*e.IdleTimeoutSecs)
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

// apiError is the error body returned by the lakebox API.
type apiError struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

func (e *apiError) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
}

func newLakeboxAPI(w *databricks.WorkspaceClient) *lakeboxAPI {
	return &lakeboxAPI{w: w}
}

// create calls POST /api/2.0/lakebox with an optional public key.
func (a *lakeboxAPI) create(ctx context.Context, publicKey string) (*createResponse, error) {
	body := createRequest{PublicKey: publicKey}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := a.doRequest(ctx, "POST", lakeboxAPIPath, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result createResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// list calls GET /api/2.0/lakebox/sandboxes.
func (a *lakeboxAPI) list(ctx context.Context) ([]sandboxEntry, error) {
	resp, err := a.doRequest(ctx, "GET", lakeboxAPIPath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result listResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return result.Sandboxes, nil
}

// get calls GET /api/2.0/lakebox/sandboxes/{id}.
func (a *lakeboxAPI) get(ctx context.Context, id string) (*sandboxEntry, error) {
	resp, err := a.doRequest(ctx, "GET", lakeboxAPIPath+"/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result sandboxEntry
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// delete calls DELETE /api/2.0/lakebox/sandboxes/{id}.
func (a *lakeboxAPI) delete(ctx context.Context, id string) error {
	resp, err := a.doRequest(ctx, "DELETE", lakeboxAPIPath+"/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseAPIError(resp)
	}
	return nil
}

// doRequest makes an authenticated HTTP request to the workspace.
func (a *lakeboxAPI) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	host := strings.TrimRight(a.w.Config.Host, "/")
	url := host + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := a.w.Config.Authenticate(req); err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(req)
}

func parseAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var apiErr apiError
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
		return &apiErr
	}
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
}

// SSH keys are now nested under the lakebox service namespace alongside
// `sandboxes/` (see `LakeboxService.CreateSshKey`).
const lakeboxKeysAPIPath = "/api/2.0/lakebox/ssh-keys"

// registerKeyRequest is the JSON body for POST /api/2.0/lakebox/ssh-keys.
type registerKeyRequest struct {
	PublicKey string `json:"public_key"`
	Name      string `json:"name,omitempty"`
}

// registerKey calls POST /api/2.0/lakebox/ssh-keys.
func (a *lakeboxAPI) registerKey(ctx context.Context, publicKey string) error {
	body := registerKeyRequest{PublicKey: publicKey}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := a.doRequest(ctx, "POST", lakeboxKeysAPIPath, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseAPIError(resp)
	}
	return nil
}
