package migrate

import (
	"context"
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
	"github.com/databricks/cli/libs/safeerr"
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
) (bool, string, error) {
	warningsSeen := false
	// PII-free description of the first warning, for telemetry. A warning stops
	// an automatic migration just as an error does, but carries no error to
	// describe it, so it is built here from the parts that are safe to report.
	warnSaferr := ""
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
			return warningsSeen, warnSaferr, err
		}
	}

	for _, node := range nodes {
		// Errors below report the key rather than the bare string, so their
		// templates name the resource type without the user's resource name.
		key := config.ResourceKey(node)

		id, ok := tfIDs[node]
		if !ok {
			// Resource is in config but not in TF state (new resource); skip.
			log.Infof(ctx, "%s: not found in terraform state, skipping", node)
			continue
		}

		group := config.GetResourceTypeFromKey(node)
		if group == "" {
			return warningsSeen, warnSaferr, safeerr.Errorf("cannot determine resource type for %q", key)
		}

		adapter, ok := adapters[group]
		if !ok {
			warningsSeen = true
			log.Warnf(ctx, warnPrefix+"unsupported resource type %q for %s, skipping", group, node)
			setWarnSaferr(&warnSaferr, safeerr.Errorf("unsupported resource type %q for %s, skipping", safeerr.Safe(group), key))
			continue
		}

		inputConfig, err := configRoot.GetResourceConfig(node)
		if err != nil {
			return warningsSeen, warnSaferr, safeerr.Errorf("%s: getting config: %w", key, err)
		}

		inputSV, err := adapter.PrepareInputConfig(inputConfig, node)
		if err != nil {
			return warningsSeen, warnSaferr, safeerr.Errorf("%s: PrepareInputConfig: %w", key, err)
		}

		newStateValue, err := adapter.PrepareState(inputSV.Value)
		if err != nil {
			return warningsSeen, warnSaferr, safeerr.Errorf("%s: PrepareState: %w", key, err)
		}

		refs, err := direct.ExtractReferences(configRoot.Value(), node, adapter.StateType())
		if err != nil {
			return warningsSeen, warnSaferr, safeerr.Errorf("%s: extracting references: %w", key, err)
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
				return warningsSeen, warnSaferr, safeerr.Errorf("%s: setting object_id: %w", key, err)
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
				return warningsSeen, warnSaferr, safeerr.Errorf("%s: parsing field path %q: %w", key, pending.fieldPathStr, err)
			}

			// ResolveFieldRef returns the fully resolved value for this field,
			// using either Method A (TF state lookup) or Method B (template evaluation).
			value, warned, err := ResolveFieldRef(ctx, tfAttrs, srcGroup, srcName, fieldPath, pending.refTemplate, warnPrefix)
			if err != nil {
				return warningsSeen, warnSaferr, safeerr.Errorf("%s: cannot resolve field %q (template %q): %w", key, pending.fieldPathStr, pending.refTemplate, err)
			}
			if warned {
				warningsSeen = true
				// The disagreeing values are user data; the resource type and the
				// stage are not.
				setWarnSaferr(&warnSaferr, safeerr.Errorf(
					"%s.%s field %q: method A and method B disagree",
					safeerr.Safe(srcGroup), srcName, pending.fieldPathStr))
			}

			// Set the resolved value directly and remove the ref entry.
			if err := structaccess.Set(sv.Value, fieldPath, value); err != nil {
				return warningsSeen, warnSaferr, safeerr.Errorf("%s: cannot set resolved value for field %q: %w", key, pending.fieldPathStr, err)
			}
			delete(sv.Refs, pending.fieldPathStr)
		}

		if len(sv.Refs) > 0 {
			return warningsSeen, warnSaferr, safeerr.Errorf("%s: unresolved references: %v", key, sv.Refs)
		}

		// Handle etag for dashboards: read it directly from TF state attributes.
		// The "etag" field is a computed TF attribute not present in the bundle config,
		// so it does not flow through PrepareState/ExtractReferences. Resources without
		// an etag return an error from LookupTFField, which we treat as "no etag".
		if v, err := LookupTFField(tfAttrs, group, srcName, structpath.NewStringKey(nil, "etag")); err == nil {
			if etag, ok := v.(string); ok && etag != "" {
				if err := structaccess.Set(sv.Value, structpath.NewStringKey(nil, "etag"), etag); err != nil {
					return warningsSeen, warnSaferr, safeerr.Errorf("%s: cannot set etag: %w", key, err)
				}
			}
		}

		if err := stateDB.SaveState(node, id, sv.Value, dependsOn); err != nil {
			return warningsSeen, warnSaferr, safeerr.Errorf("%s: SaveState: %w", key, err)
		}
	}

	return warningsSeen, warnSaferr, nil
}

// setWarnSaferr records err's message template in target unless one is already
// there: the first warning is the one reported, matching how the first error
// diagnostic is the one a deploy reports.
func setWarnSaferr(target *string, err error) {
	if *target == "" {
		*target = safeerr.SafeError(err)
	}
}
