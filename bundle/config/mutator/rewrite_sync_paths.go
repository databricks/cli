package mutator

import (
	"context"
	"maps"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/bundle"

	cv "github.com/databricks/cli/libs/config"
)

type rewriteSyncPaths struct{}

func RewriteSyncPaths() bundle.Mutator {
	return &rewriteSyncPaths{}
}

func (m *rewriteSyncPaths) Name() string {
	return "RewriteSyncPaths"
}

func (m *rewriteSyncPaths) makeRelativeTo(root string, seq cv.Value) (cv.Value, error) {
	if seq == cv.NilValue || seq.Kind() != cv.KindSequence {
		return cv.NilValue, nil
	}

	out, ok := seq.AsSequence()
	if !ok {
		return seq, nil
	}

	out = slices.Clone(out)
	for i, v := range out {
		if v.Kind() != cv.KindString {
			continue
		}

		dir := filepath.Dir(v.Location().File)
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return cv.NilValue, err
		}

		out[i] = cv.NewValue(filepath.Join(rel, v.MustString()), v.Location())
	}

	return cv.NewValue(out, seq.Location()), nil
}

func (m *rewriteSyncPaths) fn(root string) func(c cv.Value) (cv.Value, error) {
	return func(c cv.Value) (cv.Value, error) {
		var err error

		// First build a new sync object
		sync := c.Get("sync")
		if sync == cv.NilValue {
			return c, nil
		}

		out, ok := sync.AsMap()
		if !ok {
			return c, nil
		}

		out = maps.Clone(out)

		out["include"], err = m.makeRelativeTo(root, out["include"])
		if err != nil {
			return c, err
		}

		out["exclude"], err = m.makeRelativeTo(root, out["exclude"])
		if err != nil {
			return c, err
		}

		// Then replace the sync object with the new one
		return c.SetKey("sync", cv.NewValue(out, sync.Location())), nil
	}
}

func (m *rewriteSyncPaths) Apply(ctx context.Context, b *bundle.Bundle) error {
	return b.Config.Mutate(m.fn(b.Config.Path))
}
