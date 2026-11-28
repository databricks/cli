package resourcemutator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

const (
	genieSourceTable    = "table"
	genieSourceFunction = "function"
)

type skipMissingGenieDataSources struct{}

func SkipMissingGenieDataSources() bundle.Mutator {
	return &skipMissingGenieDataSources{}
}

func (m skipMissingGenieDataSources) Name() string {
	return "SkipMissingGenieDataSources"
}

func (m skipMissingGenieDataSources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	type update struct {
		space *resources.GenieSpace
		next  string
	}

	var updates []update
	var diags diag.Diagnostics
	var client *databricks.WorkspaceClient

	for key, space := range b.Config.Resources.GenieSpaces {
		if space == nil || !space.SkipMissingDataSources {
			continue
		}

		raw, ok := space.SerializedSpace.(string)
		if !ok {
			if space.SerializedSpace == nil {
				continue
			}
			return diag.Errorf("internal error: genie space %s serialized_space should have been normalized to a string, got %T", key, space.SerializedSpace)
		}
		if strings.TrimSpace(raw) == "" {
			continue
		}

		if client == nil {
			client = b.WorkspaceClient(ctx)
		}

		// A lookup failure fails the whole apply before any space is rewritten,
		// so a permission error cannot leave a partially stripped payload.
		next, spaceDiags, err := stripMissingGenieSources(ctx, client, key, raw)
		if err != nil {
			return diag.FromErr(err)
		}
		diags = diags.Extend(spaceDiags)
		if next != "" {
			updates = append(updates, update{space: space, next: next})
		}
	}

	for _, u := range updates {
		u.space.SerializedSpace = u.next
	}
	return diags
}

type genieSourceRef struct {
	kind string
	name string
}

// stripMissingGenieSources removes missing Unity Catalog objects from a
// serialized Genie space. next is empty when the payload is unchanged, so a
// space whose sources all exist keeps the original JSON bytes.
//
// example_question_sqls and text_instructions are left as authored. The Genie
// API fetches attached tables and SQL functions; it does not fetch those strings.
func stripMissingGenieSources(ctx context.Context, w *databricks.WorkspaceClient, resourceKey, raw string) (string, diag.Diagnostics, error) {
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", nil, fmt.Errorf("genie space %s serialized_space is not valid JSON: %w", resourceKey, err)
	}
	root, ok := parsed.(map[string]any)
	if !ok || root == nil {
		return "", nil, fmt.Errorf("genie space %s serialized_space must be a JSON object", resourceKey)
	}

	dataSources, _ := root["data_sources"].(map[string]any)
	instructions, _ := root["instructions"].(map[string]any)

	seen := map[string]struct{}{}
	var order []genieSourceRef
	collectGenieSources(listAt(dataSources, "tables"), genieSourceTable, seen, &order)
	collectGenieSources(listAt(dataSources, "metric_views"), genieSourceTable, seen, &order)
	collectGenieSources(listAt(instructions, "sql_functions"), genieSourceFunction, seen, &order)

	missingTables := map[string]struct{}{}
	missingFunctions := map[string]struct{}{}
	var diags diag.Diagnostics
	paths := structpath.NewPathSlice("resources", "genie_spaces", resourceKey, "serialized_space")

	for _, ref := range order {
		missing, message, err := lookupGenieSource(ctx, w, ref)
		if err != nil {
			return "", nil, err
		}
		if !missing {
			continue
		}
		if ref.kind == genieSourceFunction {
			missingFunctions[ref.name] = struct{}{}
		} else {
			missingTables[ref.name] = struct{}{}
		}
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("skipping missing Genie data source %s: %s", ref.name, message),
			Paths:    paths,
		})
	}

	if len(missingTables) == 0 && len(missingFunctions) == 0 {
		return "", diags, nil
	}

	changed := false
	if dataSources != nil {
		if filterGenieSourceList(dataSources, "tables", missingTables) {
			changed = true
		}
		if filterGenieSourceList(dataSources, "metric_views", missingTables) {
			changed = true
		}
		if len(dataSources) == 0 {
			delete(root, "data_sources")
		}
	}
	if instructions != nil {
		if filterGenieSourceList(instructions, "sql_functions", missingFunctions) {
			changed = true
		}
		if filterGenieJoinSpecs(instructions, missingTables) {
			changed = true
		}
		if len(instructions) == 0 {
			delete(root, "instructions")
		}
	}
	if !changed {
		return "", diags, nil
	}

	next, err := marshalGenieSpace(root)
	if err != nil {
		return "", nil, err
	}
	return next, diags, nil
}

// isThreePartUCName reports catalog.schema.object. Empty segments are not
// looked up; the API still sees the original name and reports the shape error.
func isThreePartUCName(name string) bool {
	parts := strings.Split(name, ".")
	return len(parts) == 3 && !slices.Contains(parts, "")
}

func collectGenieSources(list []any, kind string, seen map[string]struct{}, order *[]genieSourceRef) {
	for _, item := range list {
		name, ok := genieSourceIdentifier(item)
		// Names that are not catalog.schema.object stay in the payload so the
		// Genie API still reports the shape error.
		if !ok || !isThreePartUCName(name) {
			continue
		}
		key := kind + "\x00" + name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		*order = append(*order, genieSourceRef{kind: kind, name: name})
	}
}

func lookupGenieSource(ctx context.Context, w *databricks.WorkspaceClient, ref genieSourceRef) (bool, string, error) {
	var err error
	if ref.kind == genieSourceFunction {
		_, err = w.Functions.Get(ctx, catalog.GetFunctionRequest{Name: ref.name})
	} else {
		_, err = w.Tables.Get(ctx, catalog.GetTableRequest{FullName: ref.name})
	}
	if err == nil {
		return false, "", nil
	}
	if apierr.IsMissing(err) {
		return true, err.Error(), nil
	}
	if ref.kind == genieSourceFunction {
		return false, "", fmt.Errorf("failed to read SQL function %s: %w", ref.name, err)
	}
	return false, "", fmt.Errorf("failed to read table %s: %w", ref.name, err)
}

// filterGenieSourceList drops entries whose identifier is in missing, preserving
// the order of what remains. An emptied list is omitted.
func filterGenieSourceList(parent map[string]any, key string, missing map[string]struct{}) bool {
	list, ok := parent[key].([]any)
	if !ok {
		return false
	}
	kept := make([]any, 0, len(list))
	removed := false
	for _, item := range list {
		name, ok := genieSourceIdentifier(item)
		if ok {
			if _, skip := missing[name]; skip {
				removed = true
				continue
			}
		}
		kept = append(kept, item)
	}
	if !removed {
		return false
	}
	if len(kept) == 0 {
		delete(parent, key)
		return true
	}
	parent[key] = kept
	return true
}

func filterGenieJoinSpecs(instructions map[string]any, missingTables map[string]struct{}) bool {
	list, ok := instructions["join_specs"].([]any)
	if !ok {
		return false
	}
	kept := make([]any, 0, len(list))
	removed := false
	for _, item := range list {
		if genieJoinReferencesMissing(item, missingTables) {
			removed = true
			continue
		}
		kept = append(kept, item)
	}
	if !removed {
		return false
	}
	if len(kept) == 0 {
		delete(instructions, "join_specs")
		return true
	}
	instructions["join_specs"] = kept
	return true
}

func genieJoinReferencesMissing(item any, missingTables map[string]struct{}) bool {
	spec, ok := item.(map[string]any)
	if !ok {
		return false
	}
	for _, side := range []string{"left", "right"} {
		name, ok := genieSourceIdentifier(spec[side])
		if !ok {
			continue
		}
		if _, skip := missingTables[name]; skip {
			return true
		}
	}
	return false
}

// genieSourceIdentifier reads a three-part name from a list entry. Serialized
// spaces store objects with an identifier field. A bare string is accepted too.
func genieSourceIdentifier(item any) (string, bool) {
	switch v := item.(type) {
	case string:
		return v, v != ""
	case map[string]any:
		name, ok := v["identifier"].(string)
		return name, ok && name != ""
	default:
		return "", false
	}
}

func listAt(parent map[string]any, key string) []any {
	if parent == nil {
		return nil
	}
	list, _ := parent[key].([]any)
	return list
}

func marshalGenieSpace(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// Leave '<', '>', and '&' intact. Example SQL uses '>=', and the default
	// encoder would turn that into '\u003e'.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", fmt.Errorf("marshal serialized_space: %w", err)
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}
