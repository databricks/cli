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

const poolsBasePath = "/api/2.0/ai-training/provisioned-capacities"

func TestPoolsCommandShape(t *testing.T) {
	list := newListPoolsCommand()
	assert.Equal(t, "pools", list.Use)
	assert.NoError(t, list.Args(list, []string{}))
	assert.Error(t, list.Args(list, []string{"x"}))

	get := newGetPoolCommand()
	assert.Equal(t, "pool [POOL_ID]", get.Use)
	assert.NoError(t, get.Args(get, []string{"pool-1"}))
	assert.NoError(t, get.Args(get, []string{})) // id optional: resolved when there's one pool
	assert.Error(t, get.Args(get, []string{"a", "b"}))

	// The subcommands are wired under `air list` and `air get`.
	assert.True(t, hasSubcommand(newListCommand(), "pools"))
	assert.True(t, hasSubcommand(newGetCommand(), "pool"))
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

func TestPoolID(t *testing.T) {
	// The full resource name is reduced to the bare id; a bare id is unchanged.
	assert.Equal(t, "pool-a", poolID("provisioned-capacities/pool-a"))
	assert.Equal(t, "pool-a", poolID("pool-a"))
}

// poolsServer serves the list and get endpoints from the given bodies. The list
// body is returned for the collection path; getByID maps a bare id to its body
// (missing ids return 404 RESOURCE_DOES_NOT_EXIST).
func poolsServer(t *testing.T, listPages []string, getByID map[string]string) *httptest.Server {
	t.Helper()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == poolsBasePath:
			body := listPages[min(call, len(listPages)-1)]
			call++
			_, _ = w.Write([]byte(body))
		case strings.HasPrefix(r.URL.Path, poolsBasePath+"/"):
			id := strings.TrimPrefix(r.URL.Path, poolsBasePath+"/")
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

func TestListPoolsJSON(t *testing.T) {
	// Two pages exercise the pagination loop; usage is absent on list, as the
	// real endpoint leaves it. name arrives as the full resource name.
	page1 := `{"provisioned_capacities":[{"name":"provisioned-capacities/pool-a","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64}}],"next_page_token":"tok2"}`
	page2 := `{"provisioned_capacities":[{"name":"provisioned-capacities/pool-b","spec":{"accelerator_type":"GPU_1xH100","accelerator_count":"8"}}]}`
	srv := poolsServer(t, []string{page1, page2}, nil)

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(t.Context(), newTestWorkspaceClient(t, srv.URL))
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newListPoolsCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)

	require.NoError(t, cmd.RunE(cmd, nil))

	var got struct {
		Data poolListData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Data.Rows, 2)
	assert.Equal(t, poolRow{ID: "pool-a", AcceleratorType: "GPU_8xH100", ReservedAccelerators: 64}, got.Data.Rows[0])
	// The second row's accelerator_count arrived as a JSON string, and the id is
	// stripped from the resource name.
	assert.Equal(t, "pool-b", got.Data.Rows[1].ID)
	assert.Equal(t, int64(8), got.Data.Rows[1].ReservedAccelerators)
}

func TestListPoolsText(t *testing.T) {
	page := `{"provisioned_capacities":[{"name":"provisioned-capacities/pool-a","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64}}]}`
	srv := poolsServer(t, []string{page}, nil)

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newListPoolsCommand(), flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	require.NoError(t, cmd.RunE(cmd, nil))
	out := buf.String()
	assert.Contains(t, out, "ACCELERATOR")
	assert.Contains(t, out, "pool-a")
	assert.Contains(t, out, "GPU_8xH100")
	assert.Contains(t, out, "64")
}

func TestListPoolsTextEmpty(t *testing.T) {
	srv := poolsServer(t, []string{`{"provisioned_capacities":[]}`}, nil)

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newListPoolsCommand(), flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	require.NoError(t, cmd.RunE(cmd, nil))
	assert.Contains(t, buf.String(), "No GPU pools found.")
}

func TestGetPoolJSON(t *testing.T) {
	body := `{"name":"provisioned-capacities/pool-a","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64},"status":{"usage":{"used_accelerator_count":40,"idle_accelerator_count":24}}}`
	srv := poolsServer(t, nil, map[string]string{"pool-a": body})

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(t.Context(), newTestWorkspaceClient(t, srv.URL))
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newGetPoolCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)

	require.NoError(t, cmd.RunE(cmd, []string{"pool-a"}))

	var got struct {
		Data poolDetailData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "pool-a", got.Data.ID)
	assert.Equal(t, "GPU_8xH100", got.Data.AcceleratorType)
	assert.Equal(t, int64(64), got.Data.ReservedAccelerators)
	require.NotNil(t, got.Data.UsedAccelerators)
	assert.Equal(t, int64(40), *got.Data.UsedAccelerators)
	require.NotNil(t, got.Data.IdleAccelerators)
	assert.Equal(t, int64(24), *got.Data.IdleAccelerators)
}

func TestGetPoolAcceptsResourceName(t *testing.T) {
	// A full resource name argument resolves to the same GET path as the bare id.
	body := `{"name":"provisioned-capacities/pool-a","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64},"status":{"usage":{"used_accelerator_count":1,"idle_accelerator_count":63}}}`
	srv := poolsServer(t, nil, map[string]string{"pool-a": body})

	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newGetPoolCommand(), flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&bytes.Buffer{})

	require.NoError(t, cmd.RunE(cmd, []string{"provisioned-capacities/pool-a"}))
}

func TestGetPoolText(t *testing.T) {
	// A pool whose usage block is absent shows N/A for the usage cells rather
	// than a misleading zero.
	body := `{"name":"provisioned-capacities/pool-a","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64}}`
	srv := poolsServer(t, nil, map[string]string{"pool-a": body})

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newGetPoolCommand(), flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	require.NoError(t, cmd.RunE(cmd, []string{"pool-a"}))
	out := buf.String()
	assert.Contains(t, out, "Pool ID:                 pool-a")
	assert.Contains(t, out, "Reserved Accelerators:   64")
	assert.Contains(t, out, "Used Accelerators:       N/A")
}

func TestGetPoolOmitIDResolvesSolePool(t *testing.T) {
	// With exactly one pool, `air get pool` (no id) resolves it and shows usage.
	list := `{"provisioned_capacities":[{"name":"provisioned-capacities/pool-only","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64}}]}`
	detail := `{"name":"provisioned-capacities/pool-only","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64},"status":{"usage":{"used_accelerator_count":40,"idle_accelerator_count":24}}}`
	srv := poolsServer(t, []string{list}, map[string]string{"pool-only": detail})

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newGetPoolCommand(), flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	require.NoError(t, cmd.RunE(cmd, nil))
	out := buf.String()
	assert.Contains(t, out, "Pool ID:                 pool-only")
	assert.Contains(t, out, "Used Accelerators:       40")
}

func TestGetPoolOmitIDNoPools(t *testing.T) {
	srv := poolsServer(t, []string{`{"provisioned_capacities":[]}`}, nil)

	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newGetPoolCommand(), flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&bytes.Buffer{})

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no GPU pools found")
}

func TestGetPoolOmitIDMultipleErrors(t *testing.T) {
	// More than one pool: the id can't be inferred, so the error lists the ids.
	list := `{"provisioned_capacities":[` +
		`{"name":"provisioned-capacities/pool-a","spec":{"accelerator_type":"GPU_8xH100","accelerator_count":64}},` +
		`{"name":"provisioned-capacities/pool-b","spec":{"accelerator_type":"GPU_1xH100","accelerator_count":8}}]}`
	srv := poolsServer(t, []string{list}, nil)

	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(newGetPoolCommand(), flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&bytes.Buffer{})

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "specify one of: pool-a, pool-b")
}

func TestGetPoolEmptyID(t *testing.T) {
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, "https://example.com"))
	cmd := withOutput(newGetPoolCommand(), flags.OutputText)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"   "})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestListPoolsFeatureDisabledJSON(t *testing.T) {
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
	cmd := withOutput(newListPoolsCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "FEATURE_DISABLED", got.Error.Code)
	assert.False(t, got.Error.Retryable)
	assert.Contains(t, got.Error.Message, "not enabled for this workspace")
}

func TestGetPoolNotFoundJSON(t *testing.T) {
	srv := poolsServer(t, nil, map[string]string{})

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(t.Context(), newTestWorkspaceClient(t, srv.URL))
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newGetPoolCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"missing"})
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "NOT_FOUND", got.Error.Code)
	assert.Contains(t, got.Error.Message, `GPU pool "missing" not found`)
}

func TestGetPoolPlain404(t *testing.T) {
	// A bare HTTP 404 (no RESOURCE_DOES_NOT_EXIST error code) must still map to
	// NOT_FOUND, since apierr.ErrNotFound covers any 404.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"no such thing"}`))
	}))
	t.Cleanup(srv.Close)

	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(t.Context(), newTestWorkspaceClient(t, srv.URL))
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(newGetPoolCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, []string{"pool-x"})
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "NOT_FOUND", got.Error.Code)
}
