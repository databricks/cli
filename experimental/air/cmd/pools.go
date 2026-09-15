package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/spf13/cobra"
)

// A "GPU pool" is a pre-provisioned AI Runtime accelerator reservation — the
// same reservation a run targets via compute.provisioned_capacity_id. "GPU
// pool" (or "pool") is the product-facing name used across the Compute page and
// this CLI. The backend still models and serves these as AiWorkflowService
// ProvisionedCapacity resources, so the endpoint path and the wire structs
// below keep the API's name; only the user-facing surface says "pool".
const poolsAPIPath = "/api/2.0/ai-training/provisioned-capacities"

// resourceNamePrefix is the AIP resource-name prefix on ProvisionedCapacity.name
// ("provisioned-capacities/{id}"). The CLI shows and accepts the bare id.
const resourceNamePrefix = "provisioned-capacities/"

// poolListPageSize is the per-request page size for the list endpoint. A
// workspace holds very few pools, so this is effectively a single page.
const poolListPageSize = 100

// poolListMaxPages caps pagination so a misbehaving next_page_token can't loop
// forever.
const poolListMaxPages = 50

// apiInt64 decodes an int64 that the REST gateway may send either as a JSON
// number or, per proto3 JSON, as a quoted string.
type apiInt64 int64

func (n *apiInt64) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*n = apiInt64(v)
	return nil
}

// The wire structs below mirror the ProvisionedCapacity proto's JSON (the
// backend name for a GPU pool).
type provisionedCapacity struct {
	Name   string                     `json:"name"`
	Spec   *provisionedCapacitySpec   `json:"spec"`
	Status *provisionedCapacityStatus `json:"status"`
}

type provisionedCapacitySpec struct {
	AcceleratorType  string   `json:"accelerator_type"`
	AcceleratorCount apiInt64 `json:"accelerator_count"`
}

type provisionedCapacityStatus struct {
	Usage *provisionedCapacityUsage `json:"usage"`
}

type provisionedCapacityUsage struct {
	UsedAcceleratorCount apiInt64 `json:"used_accelerator_count"`
	IdleAcceleratorCount apiInt64 `json:"idle_accelerator_count"`
}

type listProvisionedCapacitiesResponse struct {
	ProvisionedCapacities []provisionedCapacity `json:"provisioned_capacities"`
	NextPageToken         string                `json:"next_page_token"`
}

// poolListData is the `air list pools` payload. Usage counts are intentionally
// absent: the list endpoint does not populate them (they come from `air get`).
type poolListData struct {
	Rows []poolRow `json:"pools"`
}

type poolRow struct {
	ID                   string `json:"pool_id"`
	AcceleratorType      string `json:"accelerator_type"`
	ReservedAccelerators int64  `json:"reserved_accelerators"`
}

// poolDetailData is the `air get pool` payload. Usage is a pointer because it is
// populated only when the pool reports it.
type poolDetailData struct {
	ID                   string `json:"pool_id"`
	AcceleratorType      string `json:"accelerator_type"`
	ReservedAccelerators int64  `json:"reserved_accelerators"`
	UsedAccelerators     *int64 `json:"used_accelerators"`
	IdleAccelerators     *int64 `json:"idle_accelerators"`
}

// poolID strips the AIP resource-name prefix, returning the bare id. Input may
// already be the bare id (from a user argument) or the full resource name (from
// a response's name field).
func poolID(name string) string {
	return strings.TrimPrefix(name, resourceNamePrefix)
}

// poolAPIError classifies a pools call failure into the CLI's error envelope,
// matching how `air get` classifies run lookups.
func poolAPIError(ctx context.Context, cmd *cobra.Command, action string, err error) error {
	// The handler gates the API behind a SAFE flag and reports FEATURE_DISABLED
	// where it is not yet rolled out. That is a permanent state for the
	// workspace, not a retryable failure — and its explicit error code is more
	// specific than the generic 403 it may arrive as, so check it first.
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.ErrorCode == "FEATURE_DISABLED" {
		return renderError(ctx, cmd, "FEATURE_DISABLED", "PERMANENT", false,
			errors.New("the GPU pools API is not enabled for this workspace"))
	}
	if errors.Is(err, apierr.ErrUnauthenticated) || errors.Is(err, apierr.ErrPermissionDenied) {
		return authError(ctx, cmd, err)
	}
	return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
		fmt.Errorf("failed to %s: %w", action, err))
}

// listPools pages the list endpoint fully.
func listPools(ctx context.Context, w *databricks.WorkspaceClient) ([]provisionedCapacity, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	var out []provisionedCapacity
	pageToken := ""
	for range poolListMaxPages {
		// GET query params ride the request arg (the SDK serializes them for a
		// GET), matching the sibling workflows call.
		query := map[string]any{"page_size": poolListPageSize}
		if pageToken != "" {
			query["page_token"] = pageToken
		}
		var resp listProvisionedCapacitiesResponse
		// WorkspaceIDHeaders is required so unified hosts route the call to the
		// caller's workspace rather than relying on Config.WorkspaceID alone.
		if err := apiClient.Do(ctx, http.MethodGet, poolsAPIPath, auth.WorkspaceIDHeaders(w.Config), nil, query, &resp); err != nil {
			return nil, err
		}
		out = append(out, resp.ProvisionedCapacities...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return out, nil
}

// getPool fetches one pool, including its usage summary.
func getPool(ctx context.Context, w *databricks.WorkspaceClient, id string) (*provisionedCapacity, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	var pc provisionedCapacity
	// WorkspaceIDHeaders is required so unified hosts route the call to the
	// caller's workspace rather than relying on Config.WorkspaceID alone.
	if err := apiClient.Do(ctx, http.MethodGet, poolsAPIPath+"/"+id, auth.WorkspaceIDHeaders(w.Config), nil, nil, &pc); err != nil {
		return nil, err
	}
	return &pc, nil
}

func newListPoolsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pools",
		Args:  root.NoArgs,
		Short: "List the GPU pools available to the current workspace",
	}

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		err := root.MustWorkspaceClient(cmd, args)
		if err == nil || errors.Is(err, root.ErrAlreadyPrinted) {
			return err
		}
		return authError(cmd.Context(), cmd, err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		pools, err := listPools(ctx, w)
		if err != nil {
			return poolAPIError(ctx, cmd, "list GPU pools", err)
		}

		data := poolListData{Rows: make([]poolRow, 0, len(pools))}
		for _, p := range pools {
			data.Rows = append(data.Rows, poolRowFrom(p))
		}

		if root.OutputType(cmd) != flags.OutputText {
			return renderEnvelope(ctx, data)
		}
		renderPoolTable(cmd.OutOrStdout(), data.Rows)
		return nil
	}

	return cmd
}

func newGetPoolCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool POOL_ID",
		Args:  root.ExactArgs(1),
		Short: "Show a GPU pool, including its accelerator usage",
	}

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		err := root.MustWorkspaceClient(cmd, args)
		if err == nil || errors.Is(err, root.ErrAlreadyPrinted) {
			return err
		}
		return authError(cmd.Context(), cmd, err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		// Accept either the bare id or the full resource name.
		id := poolID(strings.TrimSpace(args[0]))
		if id == "" {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				errors.New("pool_id cannot be empty"))
		}

		pool, err := getPool(ctx, w, id)
		if err != nil {
			// ErrNotFound covers a plain 404 as well as RESOURCE_DOES_NOT_EXIST.
			if errors.Is(err, apierr.ErrNotFound) {
				return renderError(ctx, cmd, "NOT_FOUND", "NOT_FOUND", false,
					fmt.Errorf("GPU pool %q not found: check the id with `air list pools`", id))
			}
			return poolAPIError(ctx, cmd, fmt.Sprintf("get GPU pool %q", id), err)
		}

		data := poolDetailFrom(*pool)
		if root.OutputType(cmd) != flags.OutputText {
			return renderEnvelope(ctx, data)
		}
		renderPoolDetail(cmd.OutOrStdout(), data)
		return nil
	}

	return cmd
}

// poolRowFrom projects a wire ProvisionedCapacity to a list row.
func poolRowFrom(p provisionedCapacity) poolRow {
	row := poolRow{ID: poolID(p.Name)}
	if p.Spec != nil {
		row.AcceleratorType = p.Spec.AcceleratorType
		row.ReservedAccelerators = int64(p.Spec.AcceleratorCount)
	}
	return row
}

// poolDetailFrom projects a wire ProvisionedCapacity to the detail payload.
func poolDetailFrom(p provisionedCapacity) poolDetailData {
	data := poolDetailData{ID: poolID(p.Name)}
	if p.Spec != nil {
		data.AcceleratorType = p.Spec.AcceleratorType
		data.ReservedAccelerators = int64(p.Spec.AcceleratorCount)
	}
	if p.Status != nil && p.Status.Usage != nil {
		used, idle := int64(p.Status.Usage.UsedAcceleratorCount), int64(p.Status.Usage.IdleAcceleratorCount)
		data.UsedAccelerators = &used
		data.IdleAccelerators = &idle
	}
	return data
}

// renderPoolTable prints the list as an aligned text table.
func renderPoolTable(out io.Writer, rows []poolRow) {
	if len(rows) == 0 {
		fmt.Fprintln(out, "No GPU pools found.")
		return
	}
	fmt.Fprintf(out, "%-40s %-14s %s\n", "ID", "ACCELERATOR", "RESERVED")
	for _, r := range rows {
		fmt.Fprintf(out, "%-40s %-14s %d\n", orNA(r.ID), orNA(r.AcceleratorType), r.ReservedAccelerators)
	}
}

// renderPoolDetail prints the get view as aligned label/value lines.
func renderPoolDetail(out io.Writer, d poolDetailData) {
	line := func(label, value string) { fmt.Fprintf(out, "%-24s %s\n", label+":", value) }
	line("Pool ID", orNA(d.ID))
	line("Accelerator Type", orNA(d.AcceleratorType))
	line("Reserved Accelerators", strconv.FormatInt(d.ReservedAccelerators, 10))
	line("Used Accelerators", acceleratorCell(d.UsedAccelerators))
	line("Idle Accelerators", acceleratorCell(d.IdleAccelerators))
}

// acceleratorCell renders an optional count, showing N/A when the pool did not
// report usage.
func acceleratorCell(v *int64) string {
	if v == nil {
		return na
	}
	return strconv.FormatInt(*v, 10)
}
