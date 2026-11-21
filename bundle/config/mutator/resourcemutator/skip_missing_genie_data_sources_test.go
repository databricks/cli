package resourcemutator_test

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSkipMissingGenieDataSourcesFlagOff(t *testing.T) {
	raw := `{
		"data_sources": {"tables": [{"identifier": "main.sales.customers"}]}
	}`
	m := mocks.NewMockWorkspaceClient(t)
	b := newSkipMissingBundle(t, &resources.GenieSpace{
		SerializedSpace: raw,
	}, m.WorkspaceClient)

	diags := bundle.ApplySeq(t.Context(), b, resourcemutator.SkipMissingGenieDataSources())

	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
	assert.Equal(t, raw, b.Config.Resources.GenieSpaces["my_space"].SerializedSpace)
}

func TestSkipMissingGenieDataSourcesSkipsAndWarns(t *testing.T) {
	raw := `{
		"version": 2,
		"data_sources": {
			"tables": [
				{"identifier": "main.sales.orders", "column_configs": [{"column_name": "id"}]},
				{"identifier": "main.sales.customers"},
				{"identifier": "not_a_three_part_name"}
			],
			"metric_views": [
				{"identifier": "main.sales.revenue"},
				{"identifier": "main.sales.missing_mv"}
			]
		},
		"instructions": {
			"text_instructions": [{"id": "ti", "content": ["keep me\n"]}],
			"example_question_sqls": [{"id": "eq", "sql": ["SELECT 1 >= 0 FROM main.sales.customers"]}],
			"sql_functions": [
				{"id": "sf-keep", "identifier": "main.analytics.keep_fn"},
				{"id": "sf-drop", "identifier": "main.analytics.missing_fn"}
			],
			"join_specs": [
				{"id": "j-drop", "left": {"identifier": "main.sales.orders", "alias": "o"}, "right": {"identifier": "main.sales.customers", "alias": "c"}, "sql": ["` + "`o`.`id` = `c`.`id`" + `"]},
				{"id": "j-keep", "left": {"identifier": "main.sales.orders", "alias": "o"}, "right": {"identifier": "main.sales.revenue", "alias": "r"}, "sql": ["` + "`o`.`id` = `r`.`id`" + `"]}
			]
		}
	}`

	missing := &apierr.APIError{StatusCode: 404, ErrorCode: "NOT_FOUND", Message: "does not exist"}
	m := mocks.NewMockWorkspaceClient(t)
	expectTable(m, "main.sales.orders", &catalog.TableInfo{}, nil)
	expectTable(m, "main.sales.customers", nil, missing)
	expectTable(m, "main.sales.revenue", &catalog.TableInfo{}, nil)
	expectTable(m, "main.sales.missing_mv", nil, missing)
	expectFunction(m, "main.analytics.keep_fn", &catalog.FunctionInfo{}, nil)
	expectFunction(m, "main.analytics.missing_fn", nil, missing)

	b := newSkipMissingBundle(t, &resources.GenieSpace{
		SkipMissingDataSources: true,
		SerializedSpace:        raw,
	}, m.WorkspaceClient)

	diags := bundle.ApplySeq(t.Context(), b, resourcemutator.SkipMissingGenieDataSources())

	require.NoError(t, diags.Error())
	warnings := diags.Filter(diag.Warning)
	require.Len(t, warnings, 3)
	assert.Contains(t, warnings[0].Summary, "main.sales.customers")
	assert.Contains(t, warnings[0].Summary, "does not exist")
	assert.Contains(t, warnings[1].Summary, "main.sales.missing_mv")
	assert.Contains(t, warnings[1].Summary, "does not exist")
	assert.Contains(t, warnings[2].Summary, "main.analytics.missing_fn")
	assert.Contains(t, warnings[2].Summary, "does not exist")

	got := b.Config.Resources.GenieSpaces["my_space"].SerializedSpace.(string)
	parsed := decodeSpace(t, got)
	assert.Equal(t, []string{"main.sales.orders", "not_a_three_part_name"}, sourceIdentifiers(parsed["data_sources"].(map[string]any)["tables"]))
	assert.Equal(t, []string{"main.sales.revenue"}, sourceIdentifiers(parsed["data_sources"].(map[string]any)["metric_views"]))
	instructions := parsed["instructions"].(map[string]any)
	assert.Equal(t, []string{"main.analytics.keep_fn"}, sourceIdentifiers(instructions["sql_functions"]))
	assert.Equal(t, []string{"j-keep"}, objectIDs(instructions["join_specs"]))
	assert.Contains(t, instructions, "text_instructions")
	assert.Contains(t, instructions, "example_question_sqls")
	assert.Contains(t, got, "SELECT 1 >= 0 FROM main.sales.customers")
	assert.NotContains(t, got, `\u003e`)

	tables := parsed["data_sources"].(map[string]any)["tables"].([]any)
	assert.Contains(t, tables[0].(map[string]any), "column_configs")
}

func TestSkipMissingGenieDataSourcesPermissionError(t *testing.T) {
	raw := `{"data_sources":{"tables":[{"identifier":"main.sales.orders"}]}}`
	denied := &apierr.APIError{StatusCode: 403, ErrorCode: "PERMISSION_DENIED", Message: "SELECT denied"}
	m := mocks.NewMockWorkspaceClient(t)
	expectTable(m, "main.sales.orders", nil, denied)

	b := newSkipMissingBundle(t, &resources.GenieSpace{
		SkipMissingDataSources: true,
		SerializedSpace:        raw,
	}, m.WorkspaceClient)

	diags := bundle.ApplySeq(t.Context(), b, resourcemutator.SkipMissingGenieDataSources())

	require.Error(t, diags.Error())
	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Summary, "main.sales.orders")
	assert.Contains(t, diags[0].Summary, "SELECT denied")
	assert.Equal(t, raw, b.Config.Resources.GenieSpaces["my_space"].SerializedSpace)
}

func TestSkipMissingGenieDataSourcesSecondApplyKeepsIdentifier(t *testing.T) {
	raw := `{"version":2,"data_sources":{"tables":[{"identifier":"main.sales.orders"}]}}`
	missing := &apierr.APIError{StatusCode: 404, ErrorCode: "NOT_FOUND", Message: "does not exist"}
	m := mocks.NewMockWorkspaceClient(t)
	expectTable(m, "main.sales.orders", nil, missing)

	b := newSkipMissingBundle(t, &resources.GenieSpace{
		SkipMissingDataSources: true,
		SerializedSpace:        raw,
	}, m.WorkspaceClient)

	diags := bundle.ApplySeq(t.Context(), b, resourcemutator.SkipMissingGenieDataSources())
	require.NoError(t, diags.Error())
	assert.NotContains(t, b.Config.Resources.GenieSpaces["my_space"].SerializedSpace.(string), "main.sales.orders")

	// Each plan loads the bundle definition again. The reduced payload is what
	// was stored in state, not what the next apply starts from.
	expectTable(m, "main.sales.orders", &catalog.TableInfo{}, nil)
	b = newSkipMissingBundle(t, &resources.GenieSpace{
		SkipMissingDataSources: true,
		SerializedSpace:        raw,
	}, m.WorkspaceClient)

	diags = bundle.ApplySeq(t.Context(), b, resourcemutator.SkipMissingGenieDataSources())
	require.NoError(t, diags.Error())
	assert.Empty(t, diags.Filter(diag.Warning))
	assert.Equal(t, raw, b.Config.Resources.GenieSpaces["my_space"].SerializedSpace)
}

func newSkipMissingBundle(t *testing.T, space *resources.GenieSpace, w *databricks.WorkspaceClient) *bundle.Bundle {
	t.Helper()
	dir := t.TempDir()
	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Resources: config.Resources{
				GenieSpaces: map[string]*resources.GenieSpace{"my_space": space},
			},
		},
	}
	if w != nil {
		b.SetWorkpaceClient(w)
	}
	return b
}

func expectTable(m *mocks.MockWorkspaceClient, fullName string, info *catalog.TableInfo, err error) {
	m.GetMockTablesAPI().EXPECT().
		Get(mock.Anything, catalog.GetTableRequest{FullName: fullName}).
		Return(info, err).
		Once()
}

func expectFunction(m *mocks.MockWorkspaceClient, name string, info *catalog.FunctionInfo, err error) {
	m.GetMockFunctionsAPI().EXPECT().
		Get(mock.Anything, catalog.GetFunctionRequest{Name: name}).
		Return(info, err).
		Once()
}

func decodeSpace(t *testing.T, raw string) map[string]any {
	t.Helper()
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))
	return parsed
}

func sourceIdentifiers(list any) []string {
	items, _ := list.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case string:
			out = append(out, v)
		case map[string]any:
			name, _ := v["identifier"].(string)
			out = append(out, name)
		}
	}
	return out
}

func objectIDs(list any) []string {
	items, _ := list.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		obj, _ := item.(map[string]any)
		id, _ := obj["id"].(string)
		out = append(out, id)
	}
	return out
}
