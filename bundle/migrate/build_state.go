package migrate

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

// BuildStateFromTF iterates over bundle resources, resolves cross-resource
// references using TF state attributes, and writes each resource's state entry.
// configRoot should be an un-interpolated config (with ${resources.*} references).
func BuildStateFromTF(
	ctx context.Context,
	configRoot *config.Root,
	adapters map[string]*dresources.Adapter,
	stateDB *dstate.DeploymentState,
	tfAttrs TFStateAttrs,
	tfIDs map[string]string,
	warnPrefix string,
) (bool, error) {
	warningsSeen := false
	// Collect all resource nodes (same patterns as makePlan).
	var nodes []string
	patterns := []dyn.Pattern{
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key("permissions")),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key("grants")),
	}
	for _, pat := range patterns {
		_, err := dyn.MapByPattern(
			configRoot.Value(),
			pat,
			func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				nodes = append(nodes, p.String())
				return dyn.InvalidValue, nil
			},
		)
		if err != nil {
			return warningsSeen, err
		}
	}

	for _, node := range nodes {
		id, ok := tfIDs[node]
		if !ok {
			// Resource is in config but not in TF state (new resource); skip.
			log.Infof(ctx, "%s: not found in terraform state, skipping", node)
			continue
		}

		group := config.GetResourceTypeFromKey(node)
		if group == "" {
			return warningsSeen, fmt.Errorf("cannot determine resource type for %q", node)
		}

		adapter, ok := adapters[group]
		if !ok {
			warningsSeen = true
			log.Warnf(ctx, warnPrefix+"unsupported resource type %q for %s, skipping", group, node)
			continue
		}

		inputConfig, err := configRoot.GetResourceConfig(node)
		if err != nil {
			return warningsSeen, fmt.Errorf("%s: getting config: %w", node, err)
		}

		inputSV, err := adapter.PrepareInputConfig(inputConfig, node)
		if err != nil {
			return warningsSeen, fmt.Errorf("%s: PrepareInputConfig: %w", node, err)
		}

		newStateValue, err := adapter.PrepareState(inputSV.Value)
		if err != nil {
			return warningsSeen, fmt.Errorf("%s: PrepareState: %w", node, err)
		}

		refs, err := direct.ExtractReferences(configRoot.Value(), node, adapter.StateType())
		if err != nil {
			return warningsSeen, fmt.Errorf("%s: extracting references: %w", node, err)
		}
		maps.Copy(refs, inputSV.Refs)

		sv := structvar.NewStructVar(newStateValue, refs)

		// Compute depends_on from cross-resource references before resolving them
		// (resolution deletes entries from the refs map).
		// Same logic as makePlan in bundle/direct/bundle_plan.go.
		var dependsOn []deployplan.DependsOnEntry //nolint:prealloc
		for _, refTemplate := range refs {
			ref, ok := dynvar.NewRef(dyn.V(refTemplate))
			if !ok {
				continue
			}
			for _, targetPath := range ref.References() {
				targetPathParsed, err := dyn.NewPathFromString(targetPath)
				if err != nil {
					continue
				}
				targetNodeDP, _ := config.GetNodeAndType(targetPathParsed)
				targetNode := targetNodeDP.String()
				fullRef := "${" + targetPath + "}"
				found := false
				for _, dep := range dependsOn {
					if dep.Node == targetNode && dep.Label == fullRef {
						found = true
						break
					}
				}
				if !found {
					dependsOn = append(dependsOn, deployplan.DependsOnEntry{
						Node:  targetNode,
						Label: fullRef,
					})
				}
			}
		}
		slices.SortFunc(dependsOn, func(a, b deployplan.DependsOnEntry) int {
			if a.Node != b.Node {
				return strings.Compare(a.Node, b.Node)
			}
			return strings.Compare(a.Label, b.Label)
		})

		// For a .permissions node, id (tfIDs[node]) is the databricks_permissions resource's
		// own ID, which is exactly the object_id (e.g. "/serving-endpoints/<id>"). Use it
		// directly: re-deriving it from the parent's TF state fails for types whose id field
		// is absent there (model_serving_endpoints, database_instances).
		if _, ok := sv.Refs["object_id"]; ok {
			if err := structaccess.Set(sv.Value, structpath.NewStringKey(nil, "object_id"), id); err != nil {
				return warningsSeen, fmt.Errorf("%s: setting object_id: %w", node, err)
			}
			delete(sv.Refs, "object_id")
		}

		// Resolve each reference using TF state.
		// node format: "resources.<group>.<name>" or "resources.<group>.<name>.permissions"
		parts := strings.SplitN(node, ".", 4)
		var srcGroup, srcName string
		if len(parts) >= 3 {
			srcGroup = parts[1]
			srcName = parts[2]
		}

		// Collect all field paths that need resolution (avoid modifying map during iteration).
		type refEntry struct {
			fieldPathStr string
			refTemplate  string
		}
		var pendingRefs []refEntry
		for fieldPathStr, refTemplate := range sv.Refs {
			pendingRefs = append(pendingRefs, refEntry{fieldPathStr, refTemplate})
		}

		for _, pending := range pendingRefs {
			fieldPath, err := structpath.ParsePath(pending.fieldPathStr)
			if err != nil {
				return warningsSeen, fmt.Errorf("%s: parsing field path %q: %w", node, pending.fieldPathStr, err)
			}

			// ResolveFieldRef returns the fully resolved value for this field,
			// using either Method A (TF state lookup) or Method B (template evaluation).
			value, warned, err := ResolveFieldRef(ctx, tfAttrs, srcGroup, srcName, fieldPath, pending.refTemplate, warnPrefix)
			if err != nil {
				return warningsSeen, fmt.Errorf("%s: cannot resolve field %q (template %q): %w", node, pending.fieldPathStr, pending.refTemplate, err)
			}
			if warned {
				warningsSeen = true
			}

			// Set the resolved value directly and remove the ref entry.
			if err := structaccess.Set(sv.Value, fieldPath, value); err != nil {
				return warningsSeen, fmt.Errorf("%s: cannot set resolved value for field %q: %w", node, pending.fieldPathStr, err)
			}
			delete(sv.Refs, pending.fieldPathStr)
		}

		if len(sv.Refs) > 0 {
			return warningsSeen, fmt.Errorf("%s: unresolved references: %v", node, sv.Refs)
		}

		// Handle etag for dashboards: read it directly from TF state attributes.
		// The "etag" field is a computed TF attribute not present in the bundle config,
		// so it does not flow through PrepareState/ExtractReferences. Resources without
		// an etag return an error from LookupTFField, which we treat as "no etag".
		if v, err := LookupTFField(tfAttrs, group, srcName, structpath.NewStringKey(nil, "etag")); err == nil {
			if etag, ok := v.(string); ok && etag != "" {
				if err := structaccess.Set(sv.Value, structpath.NewStringKey(nil, "etag"), etag); err != nil {
					return warningsSeen, fmt.Errorf("%s: cannot set etag: %w", node, err)
				}
			}
		}

		// Reconcile id-composing fields with the deployed terraform state so a pending id
		// change surfaces in the plan instead of being snapshotted as already applied. A no-op
		// for permissions/grants sub-nodes: their adapters declare no id fields.
		if err := reconcileIDFields(ctx, adapter, srcGroup, srcName, sv.Value, tfAttrs, warnPrefix); err != nil {
			return warningsSeen, fmt.Errorf("%s: reconciling id fields: %w", node, err)
		}

		// Compact hashed_fields fields so the migrated state stays small. Not needed for
		// correctness — the first plan (CalculatePlan) compacts the saved state on read.
		compacted, err := dresources.CompactState(adapter.ResourceConfig(), sv.Value)
		if err != nil {
			return warningsSeen, fmt.Errorf("%s: compacting state: %w", node, err)
		}

		if err := stateDB.SaveState(ctx, node, id, compacted, dependsOn); err != nil {
			return warningsSeen, fmt.Errorf("%s: SaveState: %w", node, err)
		}
	}

	return warningsSeen, nil
}

// reconcileIDFields aligns each id-composing field (provided_id_fields, updatable_id_fields)
// in the migrated state with the deployed terraform state. The state is otherwise seeded from
// config; for id fields, recording the current config would snapshot a pending change as
// already applied and silently drift from the backend, so:
//
//   - When config and the deployed value differ only by backend normalization (identifier
//     case, trailing slash), keep the config value: the difference is not a real change and
//     recording either side converges.
//   - Otherwise the user genuinely changed the id in config without deploying it. Record the
//     deployed value so the migrated state matches what a direct deploy of the last-applied
//     config would hold, and warn: the next plan surfaces the change as a recreate (provided-id
//     fields) or a rename (updatable-id fields).
func reconcileIDFields(ctx context.Context, adapter *dresources.Adapter, group, name string, stateValue any, tfAttrs TFStateAttrs, warnPrefix string) error {
	cfg := adapter.ResourceConfig()
	if cfg == nil {
		return nil
	}
	// A provided-id change recreates the resource; an updatable-id change renames it in place.
	for _, kind := range []struct {
		rules  []dresources.FieldRule
		action string
	}{
		{cfg.ProvidedIDFields, "recreated"},
		{cfg.UpdatableIDFields, "renamed"},
	} {
		for _, rule := range kind.rules {
			path, err := structpath.ParsePath(rule.Field.String())
			if err != nil {
				continue
			}
			configVal, err := structaccess.Get(stateValue, path)
			if err != nil {
				continue
			}
			deployedVal, err := LookupTFField(tfAttrs, group, name, path)
			if err != nil {
				continue
			}

			// Every id-composing field is a string (a name, catalog_name, storage path, ...).
			configStr, ok1 := configVal.(string)
			deployedStr, ok2 := deployedVal.(string)
			if !ok1 || !ok2 {
				return fmt.Errorf("id field %q: expected string values, got config %T and deployed %T", rule.Field.String(), configVal, deployedVal)
			}

			// UC identifier names are case-insensitive and UC strips trailing slashes from
			// storage paths, so a config value that differs from the deployed one only by that
			// normalization is not a real change: keep the config value (it converges).
			if strings.EqualFold(strings.TrimRight(configStr, "/"), strings.TrimRight(deployedStr, "/")) {
				continue
			}

			// A genuine change the user has not deployed yet. Record the deployed value so the
			// migrated state matches the last-applied config, and warn so the recreate/rename in
			// the plan that follows the migration is not a surprise.
			log.Warnf(ctx, "%s%s.%s: %s differs between config (%q) and terraform state (%q); this resource will be %s.",
				warnPrefix, group, name, rule.Field.String(), configStr, deployedStr, kind.action)
			if err := structaccess.Set(stateValue, path, deployedVal); err != nil {
				return fmt.Errorf("setting id field %q: %w", rule.Field.String(), err)
			}
		}
	}
	return nil
}
