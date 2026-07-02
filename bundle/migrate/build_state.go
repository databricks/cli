package migrate

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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
) error {
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
			return err
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
			return fmt.Errorf("cannot determine resource type for %q", node)
		}

		adapter, ok := adapters[group]
		if !ok {
			log.Warnf(ctx, "unsupported resource type %q for %s, skipping", group, node)
			continue
		}

		inputConfig, err := configRoot.GetResourceConfig(node)
		if err != nil {
			return fmt.Errorf("%s: getting config: %w", node, err)
		}

		baseRefs := map[string]string{}

		switch {
		case strings.HasSuffix(node, ".permissions"):
			var sv *structvar.StructVar
			if strings.HasPrefix(node, "resources.secret_scopes.") {
				typedConfig, ok := inputConfig.(*[]resources.SecretScopePermission)
				if !ok {
					return fmt.Errorf("%s: expected *[]resources.SecretScopePermission, got %T", node, inputConfig)
				}
				sv, err = dresources.PrepareSecretScopeAclsInputConfig(*typedConfig, node)
				if err != nil {
					return fmt.Errorf("%s: preparing secret scope ACLs config: %w", node, err)
				}
			} else {
				sv, err = dresources.PreparePermissionsInputConfig(inputConfig, node)
				if err != nil {
					return fmt.Errorf("%s: preparing permissions config: %w", node, err)
				}
			}
			inputConfig = sv.Value
			baseRefs = sv.Refs

		case strings.HasSuffix(node, ".grants"):
			sv, err := dresources.PrepareGrantsInputConfig(inputConfig, node)
			if err != nil {
				return fmt.Errorf("%s: preparing grants config: %w", node, err)
			}
			inputConfig = sv.Value
			baseRefs = sv.Refs
		}

		newStateValue, err := adapter.PrepareState(inputConfig)
		if err != nil {
			return fmt.Errorf("%s: PrepareState: %w", node, err)
		}

		refs, err := direct.ExtractReferences(configRoot.Value(), node)
		if err != nil {
			return fmt.Errorf("%s: extracting references: %w", node, err)
		}
		maps.Copy(refs, baseRefs)

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
				return fmt.Errorf("%s: setting object_id: %w", node, err)
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
				return fmt.Errorf("%s: parsing field path %q: %w", node, pending.fieldPathStr, err)
			}

			// ResolveFieldRef returns the fully resolved value for this field,
			// using either Method A (TF state lookup) or Method B (template evaluation).
			value, err := ResolveFieldRef(ctx, tfAttrs, srcGroup, srcName, fieldPath, pending.refTemplate)
			if err != nil {
				return fmt.Errorf("%s: cannot resolve field %q (template %q): %w", node, pending.fieldPathStr, pending.refTemplate, err)
			}

			// Set the resolved value directly and remove the ref entry.
			if err := structaccess.Set(sv.Value, fieldPath, value); err != nil {
				return fmt.Errorf("%s: cannot set resolved value for field %q: %w", node, pending.fieldPathStr, err)
			}
			delete(sv.Refs, pending.fieldPathStr)
		}

		if len(sv.Refs) > 0 {
			return fmt.Errorf("%s: unresolved references: %v", node, sv.Refs)
		}

		// Handle etag for dashboards: read it directly from TF state attributes.
		// The "etag" field is a computed TF attribute not present in the bundle config,
		// so it does not flow through PrepareState/ExtractReferences. Resources without
		// an etag return an error from LookupTFField, which we treat as "no etag".
		if v, err := LookupTFField(tfAttrs, group, srcName, structpath.NewStringKey(nil, "etag")); err == nil {
			if etag, ok := v.(string); ok && etag != "" {
				if err := structaccess.Set(sv.Value, structpath.NewStringKey(nil, "etag"), etag); err != nil {
					return fmt.Errorf("%s: cannot set etag: %w", node, err)
				}
			}
		}

		if err := stateDB.SaveState(node, id, sv.Value, dependsOn); err != nil {
			return fmt.Errorf("%s: SaveState: %w", node, err)
		}
	}

	return nil
}
