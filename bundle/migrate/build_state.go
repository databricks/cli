package migrate

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
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
	tfIDs terraform.ExportedResourcesMap,
	etags map[string]string,
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
		idEntry, ok := tfIDs[node]
		if !ok {
			// Resource is in config but not in TF state (new resource); skip.
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

		// Handle etag for dashboards.
		if etag := etags[node]; etag != "" {
			if err := structaccess.Set(sv.Value, structpath.NewStringKey(nil, "etag"), etag); err != nil {
				return fmt.Errorf("%s: cannot set etag: %w", node, err)
			}
		}

		if err := stateDB.SaveState(node, idEntry.ID, sv.Value, nil); err != nil {
			return fmt.Errorf("%s: SaveState: %w", node, err)
		}
	}

	return nil
}
