package validate

import (
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/walk"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

// walker holds the state of a generated validation walk (required_walk.go,
// enum_walk.go).
type walker struct {
	diags diag.Diagnostics
}

// walkConfig runs the generated walker fn over the configuration of root; refs makes
// the nodes follow the pure references.
func walkConfig(root *config.Root, refs bool, fn func(*walker, *walk.Node, *config.Root)) diag.Diagnostics {
	w := &walker{}
	n := walk.Root(root)
	if refs {
		n = walk.RootWithReferences(root)
	}
	fn(w, &n, root)
	return w.diags
}

// missing reports that the required field of n is not set.
func (w *walker) missing(n *walk.Node, field string) {
	w.diags = append(w.diags, diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   fmt.Sprintf("required field %q is not set", field),
		Locations: n.Locations().Get(),
		Paths:     []*structpath.PathNode{n.Path()},
	})
}

// enumValue reports s, the value of n, if it is not one of valid. Values with variable
// references are not resolved yet and are skipped.
func (w *walker) enumValue(n *walk.Node, s string, valid []string) {
	if slices.Contains(valid, s) || structvar.ContainsVariableReference(s) {
		return
	}
	w.diags = append(w.diags, diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   fmt.Sprintf("invalid value %q for enum field. Valid values are %v", s, valid),
		Locations: n.Locations().Get(),
		Paths:     []*structpath.PathNode{n.Path()},
	})
}
