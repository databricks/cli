package resourcemutator

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resourcedefaults"
	"github.com/databricks/cli/bundle/config/walk"
)

// defaultsWalker sets the defaults of resourcedefaults.Defaults found unset by the
// generated walker (defaults_walk.go).
type defaultsWalker struct {
	b   *bundle.Bundle
	err error
}

// unset sets the default i at n.
func (w *defaultsWalker) unset(n *walk.Node, i int) {
	if w.err == nil {
		w.err = w.b.Config.Set(n.Path(), resourcedefaults.Defaults[i].Value(&w.b.Config))
	}
}

// applyResourceDefaults sets the default values of resource fields that are not set.
func applyResourceDefaults(b *bundle.Bundle) error {
	w := &defaultsWalker{b: b}
	n := walk.RootWithReferences(&b.Config)
	defaultsConfigRoot(w, &n, &b.Config)
	return w.err
}
