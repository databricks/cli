package sandbox

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
	sdk "github.com/databricks/databricks-sdk-go/service/sandbox"
)

// Legacy SSH-key registration still uses the gateway API. Lifecycle requests
// use the public SDK independently of the gateway's sandbox catalog.
const sandboxKeysAPIPath = "/api/2.0/lakebox/ssh-keys"

func sandboxName(id string) (string, error) {
	if err := validateSandboxID(id); err != nil {
		return "", err
	}
	return "sandboxes/" + url.PathEscape(id), nil
}

// max503Attempts caps attempts when the server keeps answering 503; the
// SDK's default budget would otherwise retry for up to 5 minutes.
const max503Attempts = 3

type attempt503CounterKey struct{}

// arm503Budget attaches the request's 503 attempt counter; retries of one
// request run sequentially, so a plain *int suffices.
func arm503Budget(ctx context.Context) context.Context {
	return context.WithValue(ctx, attempt503CounterKey{}, new(int))
}

// allow503Retry consumes one unit of the request's 503 budget. Unarmed
// contexts get no retries.
func allow503Retry(ctx context.Context) bool {
	n, ok := ctx.Value(attempt503CounterKey{}).(*int)
	if !ok {
		return false
	}
	*n++
	return *n < max503Attempts
}

// sandboxDocsURL points users at the Sandbox documentation, including its
// region availability. The cloud-neutral path redirects to the reader's
// cloud-specific docs page.
const sandboxDocsURL = "https://docs.databricks.com/compute/serverless/sandbox"

// translateError replaces a 503 with a user-facing message; the gateway
// body adds nothing, so it is dropped rather than wrapped.
func translateError(err error) error {
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.StatusCode == http.StatusServiceUnavailable {
		return fmt.Errorf("the Databricks Sandbox feature is not available in your region. See %s", sandboxDocsURL)
	}
	return err
}

// orgIDHeader scopes the credential to a workspace on multi-workspace
// gateways. Without it, requests fail with "Credential was not sent or was
// of an unsupported type for this API."
const orgIDHeader = "X-Databricks-Org-Id"

// Keep the CLI's established byte limit for display labels so name
// validation and its error messages remain stable across API versions.
const maxNameBytes = 256

// validateName rejects names that exceed the wire limit (counted in bytes).
func validateName(name string) error {
	if n := len(name); n > maxNameBytes {
		return fmt.Errorf("--name is %d bytes; limit is %d (emoji and most non-ASCII characters count as 2-4 bytes each)", n, maxNameBytes)
	}
	return nil
}

// sandboxAPI wraps the SDK ApiClient with workspace-id-aware request headers.
type sandboxAPI struct {
	c         *client.DatabricksClient
	sandboxes sdk.SandboxInterface
}

// createResponse preserves the CLI's JSON output independently of the SDK's
// resource representation.
type createResponse struct {
	SandboxID   string `json:"sandboxId"`
	Status      string `json:"status"`
	GatewayHost string `json:"gatewayHost,omitempty"`
}

// sandboxEntry preserves the CLI output shape for public Sandbox resources.
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

const listPageSize = 100

// Keep the established CLI inactivity policy; the public API otherwise
// defaults to no auto-stop. Persistence stays omitted for server defaults.
const defaultAutoStopSecs = 600

// registerKeyRequest is the JSON body for POST /api/2.0/lakebox/ssh-keys.
type registerKeyRequest struct {
	PublicKey string `json:"public_key"`
	Name      string `json:"name,omitempty"`
}

// newSandboxAPI returns a sandboxAPI bound to the workspace client's config.
func newSandboxAPI(w *databricks.WorkspaceClient) (*sandboxAPI, error) {
	cfg := w.Config
	if cfg.WorkspaceID == profile.WorkspaceIDNone {
		// Config contains a mutex; use the SDK's copy operation, then normalize
		// the CLI sentinel only after loaders have resolved the copied config.
		var err error
		cfg, err = w.Config.NewWithWorkspaceHost(w.Config.Host)
		if err != nil {
			return nil, fmt.Errorf("failed to create sandbox API client: %w", err)
		}
		cfg.AccountID = w.Config.AccountID
		if err := cfg.EnsureResolved(); err != nil {
			return nil, fmt.Errorf("failed to create sandbox API client: %w", err)
		}
		cfg.WorkspaceID = ""
	}
	clientCfg, err := config.HTTPClientConfigFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox API client: %w", err)
	}
	defaultRetriable := clientCfg.ErrorRetriable
	clientCfg.ErrorRetriable = func(ctx context.Context, err error) bool {
		if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.StatusCode == http.StatusServiceUnavailable {
			return allow503Retry(ctx)
		}
		return defaultRetriable(ctx, err)
	}
	c, err := client.NewWithClient(cfg, httpclient.NewApiClient(clientCfg))
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox API client: %w", err)
	}
	return &sandboxAPI{c: c, sandboxes: sdk.NewSandbox(c)}, nil
}

// headers attaches the workspace routing identifier so multi-workspace
// gateways (e.g. SPOG hosts) can scope the credential. The
// profile.WorkspaceIDNone sentinel ("none") is treated as unset so the
// literal string never goes on the wire.
func (a *sandboxAPI) headers() map[string]string {
	wsID := a.c.Config.WorkspaceID
	if wsID == "" || wsID == profile.WorkspaceIDNone {
		return nil
	}
	return map[string]string{orgIDHeader: wsID}
}

// do issues one sandbox API request: arms the 503 retry budget, attaches
// the workspace routing headers, and translates terminal errors.
func (a *sandboxAPI) do(ctx context.Context, method, path string, request, response any) error {
	return translateError(a.c.Do(arm503Budget(ctx), method, path, a.headers(), nil, request, response))
}

// sandboxFromSDK keeps the CLI's status vocabulary and JSON field names.
func sandboxFromSDK(sb *sdk.Sandbox) *sandboxEntry {
	// Keep the existing trimmed-label and ID-fallback behavior on reads.
	entry := &sandboxEntry{SandboxID: strings.TrimPrefix(sb.Name, "sandboxes/"), Status: "Unknown", Name: strings.TrimSpace(sb.DisplayName)}
	if entry.Name == "" {
		entry.Name = entry.SandboxID
	}
	if sb.Status != nil {
		switch sb.Status.State {
		case sdk.SandboxStateSandboxStatePending:
			entry.Status = "Creating"
		case sdk.SandboxStateSandboxStateRunning:
			entry.Status = "Running"
		case sdk.SandboxStateSandboxStateStopping:
			entry.Status = "Stopping"
		case sdk.SandboxStateSandboxStateStopped:
			entry.Status = "Stopped"
		}
	}
	if sb.Spec != nil && sb.Spec.Compute != nil && sb.Spec.Compute.InactivityTimeout != nil {
		timeout := sb.Spec.Compute.InactivityTimeout
		raw, _ := timeout.MarshalJSON()
		value := strings.Trim(string(raw), `"`)
		entry.IdleTimeout = &value
		disabled := timeout.AsDuration() == 0
		entry.NoAutostop = &disabled
	}
	return entry
}

func (a *sandboxAPI) create(ctx context.Context, name string) (*createResponse, error) {
	id := "sandbox-" + uuid.New().String()
	sb, err := a.sandboxes.CreateSandbox(arm503Budget(ctx), sdk.CreateSandboxRequest{
		SandboxId: id,
		Sandbox: sdk.Sandbox{
			DisplayName: name,
			Spec:        &sdk.SandboxSpec{Compute: &sdk.ComputeSpec{InactivityTimeout: duration.New(defaultAutoStopSecs * time.Second)}},
		},
	})
	if err != nil {
		// A timed-out response can leave an accepted resource behind. Report
		// its generated ID so callers can inspect it without retrying creation.
		return nil, fmt.Errorf("sandbox %s: %w", id, translateError(err))
	}
	entry := sandboxFromSDK(sb)
	return &createResponse{SandboxID: entry.SandboxID, Status: entry.Status}, nil
}

func (a *sandboxAPI) list(ctx context.Context) ([]sandboxEntry, error) {
	var all []sandboxEntry
	iterator := a.sandboxes.ListSandboxes(ctx, sdk.ListSandboxesRequest{PageSize: listPageSize})
	for iterator.HasNext(arm503Budget(ctx)) {
		sb, err := iterator.Next(ctx)
		if err != nil {
			return nil, translateError(err)
		}
		all = append(all, *sandboxFromSDK(&sb))
	}
	return all, nil
}

func (a *sandboxAPI) get(ctx context.Context, id string) (*sandboxEntry, error) {
	name, err := sandboxName(id)
	if err != nil {
		return nil, err
	}
	sb, err := a.sandboxes.GetSandbox(arm503Budget(ctx), sdk.GetSandboxRequest{Name: name})
	if err != nil {
		return nil, translateError(err)
	}
	return sandboxFromSDK(sb), nil
}

func (a *sandboxAPI) update(ctx context.Context, id string, name *string, idleTimeoutSecs *int64, noAutostop *bool) (*sandboxEntry, error) {
	resourceName, err := sandboxName(id)
	if err != nil {
		return nil, err
	}
	req := sdk.UpdateSandboxRequest{Name: resourceName}
	if name != nil {
		req.Sandbox.DisplayName = *name
		req.Sandbox.ForceSendFields = []string{"DisplayName"}
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "display_name")
	}
	if idleTimeoutSecs != nil || noAutostop != nil {
		// Preserve the existing reset to 10m when only --no-autostop=false is set.
		secs := int64(defaultAutoStopSecs)
		if idleTimeoutSecs != nil {
			secs = *idleTimeoutSecs
		}
		if noAutostop != nil && *noAutostop {
			secs = 0
		}
		req.Sandbox.Spec = &sdk.SandboxSpec{Compute: &sdk.ComputeSpec{InactivityTimeout: duration.New(time.Duration(secs) * time.Second)}}
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "spec.compute.inactivity_timeout")
	}
	sb, err := a.sandboxes.UpdateSandbox(arm503Budget(ctx), req)
	if err != nil {
		return nil, translateError(err)
	}
	return sandboxFromSDK(sb), nil
}

func (a *sandboxAPI) delete(ctx context.Context, id string) error {
	name, err := sandboxName(id)
	if err != nil {
		return err
	}
	return translateError(a.sandboxes.DeleteSandbox(arm503Budget(ctx), sdk.DeleteSandboxRequest{Name: name}))
}

func (a *sandboxAPI) stop(ctx context.Context, id string) (*sandboxEntry, error) {
	name, err := sandboxName(id)
	if err != nil {
		return nil, err
	}
	sb, err := a.sandboxes.StopSandbox(arm503Budget(ctx), sdk.StopSandboxRequest{Name: name})
	if err != nil {
		return nil, translateError(err)
	}
	return sandboxFromSDK(sb), nil
}

func (a *sandboxAPI) start(ctx context.Context, id string) (*sandboxEntry, error) {
	current, err := a.get(ctx, id)
	if err != nil {
		return nil, err
	}
	// The public API rejects start on running or pending sandboxes; the CLI
	// keeps start idempotent and lets its existing readiness poll finish.
	if current.Status == "Running" || current.Status == "Creating" {
		return current, nil
	}
	name, err := sandboxName(id)
	if err != nil {
		return nil, err
	}
	sb, err := a.sandboxes.StartSandbox(arm503Budget(ctx), sdk.StartSandboxRequest{Name: name})
	if err != nil {
		return nil, translateError(err)
	}
	return sandboxFromSDK(sb), nil
}

// registerKey calls POST /api/2.0/lakebox/ssh-keys. An empty `name` is
// omitted so the server records "unset" rather than an explicit empty string.
// Returns the registered key, which carries the workspace's `GatewayHost` —
// this is how `register` learns the gateway without a probe `list` /
// `create` call.
func (a *sandboxAPI) registerKey(ctx context.Context, publicKey, name string) (*sshKeyEntry, error) {
	var resp sshKeyEntry
	err := a.do(ctx, http.MethodPost, sandboxKeysAPIPath, registerKeyRequest{PublicKey: publicKey, Name: name}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// sshKeyEntry is a single item in the ssh-key list response. GatewayHost
// is per-workspace (not per-key) — the server stamps the same value on
// every response so callers can learn the SSH gateway without provisioning
// a sandbox first.
type sshKeyEntry struct {
	KeyHash     string `json:"keyHash"`
	Name        string `json:"name,omitempty"`
	CreateTime  string `json:"createTime,omitempty"`
	LastUseTime string `json:"lastUseTime,omitempty"`
	GatewayHost string `json:"gatewayHost,omitempty"`
}

// listKeysResponse is the JSON body returned by GET /api/2.0/lakebox/ssh-keys.
// Per-user keys are hard-capped server-side, so the full set fits in one
// response — no pagination.
type listKeysResponse struct {
	SshKeys []sshKeyEntry `json:"sshKeys"`
}

// listKeys calls GET /api/2.0/lakebox/ssh-keys.
func (a *sandboxAPI) listKeys(ctx context.Context) ([]sshKeyEntry, error) {
	var resp listKeysResponse
	err := a.do(ctx, http.MethodGet, sandboxKeysAPIPath, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.SshKeys, nil
}

// deleteKey calls DELETE /api/2.0/lakebox/ssh-keys/{key_hash}.
func (a *sandboxAPI) deleteKey(ctx context.Context, keyHash string) error {
	return a.do(ctx, http.MethodDelete, sandboxKeysAPIPath+"/"+url.PathEscape(keyHash), nil, nil)
}
