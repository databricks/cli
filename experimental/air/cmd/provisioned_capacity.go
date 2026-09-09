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
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/spf13/cobra"
)

// A "provisioned capacity" is a pre-provisioned AI Runtime accelerator
// reservation — the same reservation a run targets via
// compute.provisioned_capacity_id. It is served by AiWorkflowService's
// purpose-built public read API (the platform tracks it internally as a
// "guaranteed capacity"; the public surface calls it "provisioned capacity").
const provisionedCapacityPath = "/api/2.0/ai-training/provisioned-capacities"

// resourceNamePrefix is the AIP resource-name prefix on ProvisionedCapacity.name
// ("provisioned-capacities/{id}"). The CLI shows and accepts the bare id.
const resourceNamePrefix = "provisioned-capacities/"

// capacityListPageSize is the per-request page size for the list endpoint. A
// workspace holds very few reservations, so this is effectively a single page.
const capacityListPageSize = 100

// capacityListMaxPages caps pagination so a misbehaving next_page_token can't
// loop forever.
const capacityListMaxPages = 50

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

// The wire structs below mirror the ProvisionedCapacity proto's JSON.
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

// capacityListData is the `air list provisioned_capacity` payload. Usage counts
// are intentionally absent: the list endpoint does not populate them (they come
// from `air get`).
type capacityListData struct {
	Rows []capacityRow `json:"provisioned_capacities"`
}

type capacityRow struct {
	ID                   string `json:"provisioned_capacity_id"`
	AcceleratorType      string `json:"accelerator_type"`
	ReservedAccelerators int64  `json:"reserved_accelerators"`
}

// capacityDetailData is the `air get provisioned_capacity` payload. Usage is a
// pointer because it is populated only when the reservation reports it.
type capacityDetailData struct {
	ID                   string `json:"provisioned_capacity_id"`
	AcceleratorType      string `json:"accelerator_type"`
	ReservedAccelerators int64  `json:"reserved_accelerators"`
	UsedAccelerators     *int64 `json:"used_accelerators"`
	IdleAccelerators     *int64 `json:"idle_accelerators"`
}

// capacityID strips the AIP resource-name prefix, returning the bare id. Input
// may already be the bare id (from a user argument) or the full resource name
// (from a response's name field).
func capacityID(name string) string {
	return strings.TrimPrefix(name, resourceNamePrefix)
}

// capacityAPIError classifies a provisioned-capacities call failure into the
// CLI's error envelope, matching how `air get` classifies run lookups.
func capacityAPIError(ctx context.Context, cmd *cobra.Command, action string, err error) error {
	// The handler gates the API behind a SAFE flag and reports FEATURE_DISABLED
	// where it is not yet rolled out. That is a permanent state for the
	// workspace, not a retryable failure — and its explicit error code is more
	// specific than the generic 403 it may arrive as, so check it first.
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.ErrorCode == "FEATURE_DISABLED" {
		return renderError(ctx, cmd, "FEATURE_DISABLED", "PERMANENT", false,
			errors.New("the provisioned capacity API is not enabled for this workspace"))
	}
	if errors.Is(err, apierr.ErrUnauthenticated) || errors.Is(err, apierr.ErrPermissionDenied) {
		return authError(ctx, cmd, err)
	}
	return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
		fmt.Errorf("failed to %s: %w", action, err))
}

// listProvisionedCapacities pages the list endpoint fully.
func listProvisionedCapacities(ctx context.Context, w *databricks.WorkspaceClient) ([]provisionedCapacity, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	var out []provisionedCapacity
	pageToken := ""
	for page := 0; page < capacityListMaxPages; page++ {
		// GET query params ride the request arg (the SDK serializes them for a
		// GET), matching the sibling workflows call.
		query := map[string]any{"page_size": capacityListPageSize}
		if pageToken != "" {
			query["page_token"] = pageToken
		}
		var resp listProvisionedCapacitiesResponse
		if err := apiClient.Do(ctx, http.MethodGet, provisionedCapacityPath, nil, nil, query, &resp); err != nil {
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

// getProvisionedCapacity fetches one reservation, including its usage summary.
func getProvisionedCapacity(ctx context.Context, w *databricks.WorkspaceClient, id string) (*provisionedCapacity, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	var pc provisionedCapacity
	if err := apiClient.Do(ctx, http.MethodGet, provisionedCapacityPath+"/"+id, nil, nil, nil, &pc); err != nil {
		return nil, err
	}
	return &pc, nil
}

func newListProvisionedCapacityCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provisioned_capacity",
		Args:  root.NoArgs,
		Short: "List the pre-provisioned AI Runtime capacity reservations for the current workspace",
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

		capacities, err := listProvisionedCapacities(ctx, w)
		if err != nil {
			return capacityAPIError(ctx, cmd, "list provisioned capacities", err)
		}

		data := capacityListData{Rows: make([]capacityRow, 0, len(capacities))}
		for _, c := range capacities {
			data.Rows = append(data.Rows, capacityRowFrom(c))
		}

		if root.OutputType(cmd) != flags.OutputText {
			return renderEnvelope(ctx, data)
		}
		renderCapacityTable(cmd.OutOrStdout(), data.Rows)
		return nil
	}

	return cmd
}

func newGetProvisionedCapacityCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provisioned_capacity PROVISIONED_CAPACITY_ID",
		Args:  root.ExactArgs(1),
		Short: "Show a pre-provisioned AI Runtime capacity reservation, including its accelerator usage",
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
		id := capacityID(strings.TrimSpace(args[0]))
		if id == "" {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				errors.New("provisioned_capacity_id cannot be empty"))
		}

		pc, err := getProvisionedCapacity(ctx, w, id)
		if err != nil {
			if errors.Is(err, apierr.ErrResourceDoesNotExist) {
				return renderError(ctx, cmd, "NOT_FOUND", "NOT_FOUND", false,
					fmt.Errorf("provisioned capacity %q not found: check the id with `air list provisioned_capacity`", id))
			}
			return capacityAPIError(ctx, cmd, fmt.Sprintf("get provisioned capacity %q", id), err)
		}

		data := capacityDetailFrom(*pc)
		if root.OutputType(cmd) != flags.OutputText {
			return renderEnvelope(ctx, data)
		}
		renderCapacityDetail(cmd.OutOrStdout(), data)
		return nil
	}

	return cmd
}

// capacityRowFrom projects a wire ProvisionedCapacity to a list row.
func capacityRowFrom(c provisionedCapacity) capacityRow {
	row := capacityRow{ID: capacityID(c.Name)}
	if c.Spec != nil {
		row.AcceleratorType = c.Spec.AcceleratorType
		row.ReservedAccelerators = int64(c.Spec.AcceleratorCount)
	}
	return row
}

// capacityDetailFrom projects a wire ProvisionedCapacity to the detail payload.
func capacityDetailFrom(c provisionedCapacity) capacityDetailData {
	data := capacityDetailData{ID: capacityID(c.Name)}
	if c.Spec != nil {
		data.AcceleratorType = c.Spec.AcceleratorType
		data.ReservedAccelerators = int64(c.Spec.AcceleratorCount)
	}
	if c.Status != nil && c.Status.Usage != nil {
		used, idle := int64(c.Status.Usage.UsedAcceleratorCount), int64(c.Status.Usage.IdleAcceleratorCount)
		data.UsedAccelerators = &used
		data.IdleAccelerators = &idle
	}
	return data
}

// renderCapacityTable prints the list as an aligned text table.
func renderCapacityTable(out io.Writer, rows []capacityRow) {
	if len(rows) == 0 {
		fmt.Fprintln(out, "No provisioned capacity reservations found.")
		return
	}
	fmt.Fprintf(out, "%-40s %-14s %s\n", "ID", "ACCELERATOR", "RESERVED")
	for _, r := range rows {
		fmt.Fprintf(out, "%-40s %-14s %d\n", orNA(r.ID), orNA(r.AcceleratorType), r.ReservedAccelerators)
	}
}

// renderCapacityDetail prints the get view as aligned label/value lines.
func renderCapacityDetail(out io.Writer, d capacityDetailData) {
	line := func(label, value string) { fmt.Fprintf(out, "%-24s %s\n", label+":", value) }
	line("Provisioned Capacity ID", orNA(d.ID))
	line("Accelerator Type", orNA(d.AcceleratorType))
	line("Reserved Accelerators", strconv.FormatInt(d.ReservedAccelerators, 10))
	line("Used Accelerators", acceleratorCell(d.UsedAccelerators))
	line("Idle Accelerators", acceleratorCell(d.IdleAccelerators))
}

// acceleratorCell renders an optional count, showing N/A when the reservation
// did not report usage.
func acceleratorCell(v *int64) string {
	if v == nil {
		return na
	}
	return strconv.FormatInt(*v, 10)
}
