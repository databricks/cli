package aircmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const capacityBasePath = "/api/2.0/ai-training/provisioned-capacities"

func TestProvisionedCapacityCommandShape(t *testing.T) {
	list := newListProvisionedCapacityCommand()
	assert.Equal(t, "provisioned_capacity", list.Use)
	assert.NoError(t, list.Args(list, []string{}))
	assert.Error(t, list.Args(list, []string{"x"}))

	get := newGetProvisionedCapacityCommand()
	assert.Equal(t, "provisioned_capacity PROVISIONED_CAPACITY_ID", get.Use)
	assert.NoError(t, get.Args(get, []string{"cap-1"}))
	assert.Error(t, get.Args(get, []string{}))

	// The subcommands are wired under `air list` and `air get`.
	assert.True(t, hasSubcommand(newListCommand(), "provisioned_capacity"))
	assert.True(t, hasSubcommand(newGetCommand(), "provisioned_capacity"))
}

func hasSubcommand(parent *cobra.Command, name string) bool {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return true
		}
	}
	return false
}

func TestAPIInt64Unmarshal(t *testing.T) {
	var s struct {
		N apiInt64 `json:"n"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{"n": 42}`), &s))
	assert.EqualValues(t, 42, s.N)

	// proto3 JSON encodes int64 as a string; accept it too.
	require.NoError(t, json.Unmarshal([]byte(`{"n": "64"}`), &s))
	assert.EqualValues(t, 64, s.N)

	// A null (or absent) field leaves the zero value in place: encoding/json
	// treats UnmarshalJSON("null") as a no-op, so a fresh struct stays 0.
	var fresh struct {
		N apiInt64 `json:"n"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{"n": null}`), &fresh))
	assert.EqualValues(t, 0, fresh.N)
}

func TestCapacityID(t *testing.T) {
	// The full resource name is reduced to the bare id; a bare id is unchanged.
	assert.Equal(t, "cap-a", capacityID("provisioned-capacities/cap-a"))
	assert.Equal(t, "cap-a", capacityID("cap-a"))
}

// capacityServer serves the list and get endpoints from the given bodies. The
// list body is returned for the collection path; getByID maps a bare id to its
// body (missing ids return 404 RESOURCE_DOES_NOT_EXIST).
func capacityServer(t *testing.T, listPages []string, getByID map[string]string) *httptest.Server {
	t.Helper()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == capacityBasePath:
			body := listPages[min(call, len(listPages)-1)]
			call++
			_, _ = w.Write([]byte(body))
		case strings.HasPrefix(r.URL.Path, capacityBasePath+"/"):
			id := strings.TrimPrefix(r.URL.Path, capacityBasePath+"/")
			body, ok := getByID[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"not found"}`))
				return
			}
			_, _ = w.Write([]byte(body))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestListProvisionedCapacityJSON(t *testing.T) {
	// Two pages exercise the pagination loop; usage is absent on list, as the
	// real endpoint leaves it. name arrives as the full resource name.
	page1 := `{"provisioned_capacities":[{"name":"provisioned-capacities/cap-a","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64}}],"next_page_token":"tok2"}`
	page2 := `{"provisioned_capacities":[{"name":"provisioned-capacities/cap-b","spec":{"accelerator_type":"GPU_1xH100","accelerator_count":"8"}}]}`
	srv := capacityServer(t, []string{page1, page2}, nil)

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(t.Context(), newTestWorkspaceClient(t, srv.URL))
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newListProvisionedCapacityCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)

	require.NoError(t, cmd.RunE(cmd, nil))

	var got struct {
		Data capacityListData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Data.Rows, 2)
	assert.Equal(t, capacityRow{ID: "cap-a", AcceleratorType: "GPU_8xH100", ReservedAccelerators: 64}, got.Data.Rows[0])
	// The second row's accelerator_count arrived as a JSON string, and the id is
	// stripped from the resource name.
	assert.Equal(t, "cap-b", got.Data.Rows[1].ID)
	assert.Equal(t, int64(8), got.Data.Rows[1].ReservedAccelerators)
}

func TestListProvisionedCapacityText(t *testing.T) {
	page := `{"provisioned_capacities":[{"name":"provisioned-capacities/cap-a","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64}}]}`
	srv := capacityServer(t, []string{page}, nil)

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newListProvisionedCapacityCommand(), flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	require.NoError(t, cmd.RunE(cmd, nil))
	out := buf.String()
	assert.Contains(t, out, "ACCELERATOR")
	assert.Contains(t, out, "cap-a")
	assert.Contains(t, out, "GPU_8xH100")
	assert.Contains(t, out, "64")
}

func TestListProvisionedCapacityTextEmpty(t *testing.T) {
	srv := capacityServer(t, []string{`{"provisioned_capacities":[]}`}, nil)

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newListProvisionedCapacityCommand(), flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	require.NoError(t, cmd.RunE(cmd, nil))
	assert.Contains(t, buf.String(), "No provisioned capacity reservations found.")
}

func TestGetProvisionedCapacityJSON(t *testing.T) {
	body := `{"name":"provisioned-capacities/cap-a","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64},"status":{"usage":{"used_accelerator_count":40,"idle_accelerator_count":24}}}`
	srv := capacityServer(t, nil, map[string]string{"cap-a": body})

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(t.Context(), newTestWorkspaceClient(t, srv.URL))
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newGetProvisionedCapacityCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)

	require.NoError(t, cmd.RunE(cmd, []string{"cap-a"}))

	var got struct {
		Data capacityDetailData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "cap-a", got.Data.ID)
	assert.Equal(t, "GPU_8xH100", got.Data.AcceleratorType)
	assert.Equal(t, int64(64), got.Data.ReservedAccelerators)
	require.NotNil(t, got.Data.UsedAccelerators)
	assert.Equal(t, int64(40), *got.Data.UsedAccelerators)
	require.NotNil(t, got.Data.IdleAccelerators)
	assert.Equal(t, int64(24), *got.Data.IdleAccelerators)
}

func TestGetProvisionedCapacityAcceptsResourceName(t *testing.T) {
	// A full resource name argument resolves to the same GET path as the bare id.
	body := `{"name":"provisioned-capacities/cap-a","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64},"status":{"usage":{"used_accelerator_count":1,"idle_accelerator_count":63}}}`
	srv := capacityServer(t, nil, map[string]string{"cap-a": body})

	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newGetProvisionedCapacityCommand(), flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&bytes.Buffer{})

	require.NoError(t, cmd.RunE(cmd, []string{"provisioned-capacities/cap-a"}))
}

func TestGetProvisionedCapacityText(t *testing.T) {
	// A reservation whose usage block is absent shows N/A for the usage cells
	// rather than a misleading zero.
	body := `{"name":"provisioned-capacities/cap-a","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64}}`
	srv := capacityServer(t, nil, map[string]string{"cap-a": body})

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newGetProvisionedCapacityCommand(), flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	require.NoError(t, cmd.RunE(cmd, []string{"cap-a"}))
	out := buf.String()
	assert.Contains(t, out, "Provisioned Capacity ID: cap-a")
	assert.Contains(t, out, "Reserved Accelerators:   64")
	assert.Contains(t, out, "Used Accelerators:       N/A")
}

func TestGetProvisionedCapacityEmptyID(t *testing.T) {
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, "https://example.com"))
	cmd := withOutput(newGetProvisionedCapacityCommand(), flags.OutputText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"   "})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestListProvisionedCapacityFeatureDisabledJSON(t *testing.T) {
	// The SAFE-gated handler reports FEATURE_DISABLED before rollout; it must be
	// a permanent, non-retryable error, not a transient one.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error_code":"FEATURE_DISABLED","message":"The provisioned capacity API is not yet enabled."}`))
	}))
	t.Cleanup(srv.Close)

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(t.Context(), newTestWorkspaceClient(t, srv.URL))
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newListProvisionedCapacityCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "FEATURE_DISABLED", got.Error.Code)
	assert.False(t, got.Error.Retryable)
	assert.Contains(t, got.Error.Message, "not enabled for this workspace")
}

func TestGetProvisionedCapacityNotFoundJSON(t *testing.T) {
	srv := capacityServer(t, nil, map[string]string{})

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(t.Context(), newTestWorkspaceClient(t, srv.URL))
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newGetProvisionedCapacityCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"missing"})
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "NOT_FOUND", got.Error.Code)
	assert.Contains(t, got.Error.Message, `provisioned capacity "missing" not found`)
}
