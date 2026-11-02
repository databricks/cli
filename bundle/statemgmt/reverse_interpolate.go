package statemgmt

import (
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/walk"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

// reverseInterpolate converts terraform-style resource references in s to bundle-style.
// Example: ${databricks_pipeline.my_etl.id} → ${resources.pipelines.my_etl.id}
func reverseInterpolate(s string) (string, error) {
	out, err := structvar.Resolve(map[string]structvar.Template{"": {Value: s}}, func(sp *structpath.PathNode) (structvar.View, error) {
		// Need at least 2 components: resource_type.resource_name
		if sp.Len() < 2 {
			return structvar.View{}, structvar.ErrSkipResolution
		}

		resourceType := sp.KeyAt(0)
		isAlreadyBundleFormat := resourceType == "resources"
		if isAlreadyBundleFormat {
			return structvar.View{}, structvar.ErrSkipResolution
		}

		bundleGroup, ok := terraform.TerraformToGroupName[resourceType]
		if !ok {
			return structvar.View{}, structvar.ErrSkipResolution
		}

		// Reconstruct path in bundle format:
		// databricks_pipeline.my_pipeline.id → resources.pipelines.my_pipeline.id
		ref := fmt.Sprintf("${resources.%s.%s}", bundleGroup, sp.SkipPrefix(1))
		return structvar.NewView(&ref, nil, nil), nil
	})
	if err != nil {
		return "", err
	}
	if v, ok := out[""]; ok {
		rs, _ := v.AsString()
		return rs, nil
	}
	return s, nil
}

// reverseInterpolateConfig returns a copy of cfg in which terraform-style references
// are rewritten to bundle-style ones (see [reverseInterpolate]). cfg is not changed.
func reverseInterpolateConfig(cfg *config.Root) (*config.Root, error) {
	out := &config.Root{}
	if err := out.Assign(nil, cfg.View()); err != nil {
		return nil, err
	}

	var u reverseUpdates
	walk.Strings(cfg, walk.WithReferences, func(n *walk.Node, s string) (string, bool) {
		u.add(n.Path, s)
		return "", false
	})
	for p, ref := range cfg.References() {
		u.add(func() *structpath.PathNode { return p }, ref)
	}
	if u.err != nil {
		return nil, u.err
	}

	for _, up := range u.updates {
		if err := out.SetReference(up.path, up.s); err != nil {
			return nil, err
		}
	}
	return out, nil
}

type reverseUpdate struct {
	path *structpath.PathNode
	s    string
}

// reverseUpdates collects the strings of a configuration that change when reverse
// interpolated, and the first error.
type reverseUpdates struct {
	updates []reverseUpdate
	err     error
}

// add records the reverse interpolation of s, a value at path, if it has references
// and changes. path is only called then.
func (u *reverseUpdates) add(path func() *structpath.PathNode, s string) {
	if u.err != nil {
		return
	}
	if _, ok := structvar.NewRef(s); !ok {
		return
	}
	rs, err := reverseInterpolate(s)
	if err != nil {
		u.err = err
		return
	}
	if rs != s {
		u.updates = append(u.updates, reverseUpdate{path(), rs})
	}
}
