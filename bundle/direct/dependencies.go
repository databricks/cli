package direct

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// configNodes returns the sorted keys of the resources, permissions and grants in configRoot.
func configNodes(configRoot *config.Root) ([]string, error) {
	if configRoot == nil {
		return nil, nil
	}

	patterns := []dyn.Pattern{
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key("permissions")),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key("grants")),
	}

	var nodes []string
	for _, pat := range patterns {
		_, err := dyn.MapByPattern(
			configRoot.Value(),
			pat,
			func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				s := p.String()
				resourceType := config.GetResourceTypeFromKey(s)
				if resourceType == "" {
					return v, fmt.Errorf("cannot parse resource key: %q", s)
				}
				_, ok := dresources.SupportedResources[resourceType]
				if !ok {
					return v, fmt.Errorf("unsupported resource type: %s", resourceType)
				}

				nodes = append(nodes, s)
				return dyn.InvalidValue, nil
			},
		)
		if err != nil {
			return nil, err
		}
	}

	// MapByPattern visits keys in random order.
	slices.Sort(nodes)
	return nodes, nil
}

// nodeDependsOn returns the sorted dependencies of a node: the resources its references in refs
// point to, and its lifecycle.depends_on entries.
func nodeDependsOn(configRoot *config.Root, inputConfig any, refs map[string]string) ([]deployplan.DependsOnEntry, error) {
	var dependsOn []deployplan.DependsOnEntry
	for _, reference := range refs {
		ref, ok := dynvar.NewRef(dyn.V(reference))
		if !ok {
			continue
		}

		for _, targetPath := range ref.References() {
			targetPathParsed, err := dyn.NewPathFromString(targetPath)
			if err != nil {
				return nil, fmt.Errorf("parsing %q: %w", targetPath, err)
			}

			targetNodeDP, _ := config.GetNodeAndType(targetPathParsed)
			entry := deployplan.DependsOnEntry{
				Node:  targetNodeDP.String(),
				Label: "${" + targetPath + "}",
			}
			if !slices.Contains(dependsOn, entry) {
				dependsOn = append(dependsOn, entry)
			}
		}
	}

	lifecycleDeps, err := LifecycleDependsOn(configRoot, inputConfig)
	if err != nil {
		return nil, err
	}
	for _, dep := range lifecycleDeps {
		if !slices.Contains(dependsOn, dep) {
			dependsOn = append(dependsOn, dep)
		}
	}

	slices.SortFunc(dependsOn, func(a, b deployplan.DependsOnEntry) int {
		if a.Node != b.Node {
			return strings.Compare(a.Node, b.Node)
		}
		return strings.Compare(a.Label, b.Label)
	})
	return dependsOn, nil
}

// DetectDependencyCycle reports a cycle in the dependencies between the resources in configRoot,
// built the way the plan builds them, without reading state or the workspace. It finds only
// cycles the plan rejects too: a node the plan may leave out (an empty one, or one it cannot
// read) and edges to nodes outside the config are left out here, so the graph is a subgraph of
// the plan's. It returns nil when there is no cycle or the config cannot be read; the plan
// reports read errors itself.
func DetectDependencyCycle(configRoot *config.Root) *dagrun.CycleError {
	nodes, err := configNodes(configRoot)
	if err != nil || len(nodes) == 0 {
		return nil //nolint:nilerr // the plan reports this error
	}

	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return nil //nolint:nilerr // the plan reports this error
	}
	b := DeploymentBundle{Adapters: adapters}

	deps := make(map[string][]deployplan.DependsOnEntry, len(nodes))
	for _, node := range nodes {
		nodeDeps, ok := configNodeDependsOn(&b, configRoot, node)
		if ok {
			deps[node] = nodeDeps
		}
	}

	g := dagrun.NewGraph()
	for _, node := range nodes {
		if _, ok := deps[node]; ok {
			g.AddNode(node)
		}
	}
	for _, node := range nodes {
		for _, dep := range deps[node] {
			if _, ok := deps[dep.Node]; ok {
				g.AddDirectedEdge(dep.Node, node, dep.Label)
			}
		}
	}

	if cycleErr, ok := errors.AsType[*dagrun.CycleError](g.DetectCycle()); ok {
		return cycleErr
	}
	return nil
}

// configNodeDependsOn returns the dependencies makePlan computes for a new node, and false when
// makePlan would fail on the node or skip it as empty.
func configNodeDependsOn(b *DeploymentBundle, configRoot *config.Root, node string) ([]deployplan.DependsOnEntry, bool) {
	inputConfig, err := configRoot.GetResourceConfig(node)
	if err != nil || inputConfig == nil {
		return nil, false
	}
	adapter, err := b.getAdapterForKey(node)
	if err != nil {
		return nil, false
	}
	inputStructVar, err := adapter.PrepareInputConfig(inputConfig, node)
	if err != nil {
		return nil, false
	}
	newStateConfig, err := adapter.PrepareState(inputStructVar.Value)
	if err != nil {
		return nil, false
	}
	empty, err := adapter.IsEmptyState(newStateConfig)
	if err != nil || empty {
		return nil, false
	}
	refs, err := extractReferences(configRoot.Value(), node, adapter.StateType())
	if err != nil {
		return nil, false
	}
	maps.Copy(refs, inputStructVar.Refs)
	dependsOn, err := nodeDependsOn(configRoot, inputConfig, refs)
	if err != nil {
		return nil, false
	}
	return dependsOn, true
}
