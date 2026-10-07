package direct

import (
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

var resourcesPrefix = structpath.MustParsePath("resources")

// ResolveConfigAgainstState resolves ${resources.*} references within the resource at
// target so its runner ("bundle run") sees concrete values rather than references. For a
// reference resources.<group>.<name>.<field>, the value is taken from that resource's
// persisted state (the source of truth after deploy) when available, falling back to
// the config for anything not in state. Non-resource references are left untouched, and
// references that resolve to neither are left as-is rather than failing the command.
//
// Only the target resource is resolved: its runner is the sole consumer of resolved
// config, so resolving other resources is unnecessary work. It is also unsafe — an
// unrelated resource may reference a resource that was never deployed, whose ${...}.id
// falls back to an empty config string and then fails int type-checking.
//
// The state DB holds fields like the immutable snapshot's full_path that never reach the
// config, so it must be open.
func (b *DeploymentBundle) ResolveConfigAgainstState(cfg *config.Root, target *structpath.PathNode) error {
	view := cfg.View()
	resource := view.Lookup(target)
	if !resource.IsValid() {
		return fmt.Errorf("resource %s not found in configuration", target)
	}

	// Each string is resolved on its own (references only point at other paths), then
	// written back; collect first because writing changes the view.
	type update struct {
		path  *structpath.PathNode
		value any
	}
	var updates []update

	lookup := func(path *structpath.PathNode) (structvar.View, error) {
		if !path.HasPrefix(resourcesPrefix) {
			return structvar.View{}, structvar.ErrSkipResolution
		}
		if v, ok := b.lookupStateField(path); ok {
			return v, nil
		}
		// Fall back to the config, including fields that are implied (not explicitly set).
		if v := view.Lookup(path); v.IsValid() {
			return v, nil
		}
		got, err := structaccess.Get(cfg, path)
		if err != nil {
			return structvar.View{}, structvar.ErrSkipResolution
		}
		return structvar.NewView(&got, nil, nil), nil
	}

	err := structvar.Walk(resource, func(p *structpath.PathNode, v structvar.View) error {
		s, ok := v.AsString()
		if !ok {
			return nil
		}
		if _, ok := structvar.NewRef(s); !ok {
			return nil
		}
		key := p.String()
		out, err := structvar.Resolve(map[string]structvar.Template{key: {Value: s}}, lookup)
		if err != nil {
			return err
		}
		if resolved, ok := out[key]; ok {
			updates = append(updates, update{structpath.Join(target, p.AsSlice()...), resolved.AsAny()})
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, u := range updates {
		// Decode converts like loading did, e.g. a string id into an int job_id.
		diags, err := cfg.Decode(u.path, structvar.NewView(&u.value, nil, nil))
		if err != nil {
			return err
		}
		if diags.HasError() {
			return diags.Error()
		}
	}
	return nil
}

// lookupStateField returns the value at resources.<group>.<name>.<field...> from the
// resource's persisted state, if that resource is in state and holds the field.
func (b *DeploymentBundle) lookupStateField(path *structpath.PathNode) (structvar.View, bool) {
	if path.Len() < 4 || path.KeyAt(0) != "resources" {
		return structvar.View{}, false
	}
	resourceKey := "resources." + path.KeyAt(1) + "." + path.KeyAt(2)
	entry, ok := b.StateDB.GetResourceEntry(resourceKey)
	if !ok || len(entry.State) == 0 {
		return structvar.View{}, false
	}
	// ParseJSON keeps large ids exact.
	node, err := structvar.ParseJSON(resourceKey, entry.State)
	if err != nil {
		return structvar.View{}, false
	}
	var state any
	if _, _, err := structvar.DecodeYAMLNode(resourceKey, node, &state, nil); err != nil {
		return structvar.View{}, false
	}
	v := structvar.NewView(&state, nil, nil).Lookup(path.SkipPrefix(3))
	if !v.IsValid() {
		return structvar.View{}, false
	}
	return v, true
}
