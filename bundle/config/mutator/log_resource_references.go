package mutator

import (
	"context"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// Longest field name:
// >>> len('hard_deletion_sync_min_interval_in_seconds') == 42
const maxFieldLength = 42

const maxMetricLength = 200

// logResourceReferences scans resources for ${resources.*} references and logs them.
func LogResourceReferences() bundle.Mutator {
	return &logResourceReferences{}
}

type logResourceReferences struct{}

func (m *logResourceReferences) Name() string {
	return "LogResourceReferences"
}

func (m *logResourceReferences) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	used := map[string]struct{}{}
	resources := b.Config.Value().Get("resources")
	if !resources.IsValid() {
		// No resources section, nothing to do
		return nil
	}

	_ = dyn.WalkReadOnly(resources, func(path dyn.Path, val dyn.Value) error {
		ref, ok := dynvar.NewRef(val)
		if !ok {
			return nil
		}
		for _, r := range ref.References() {
			// Only track ${resources.*} references.
			if !strings.HasPrefix(r, "resources.") {
				continue
			}

			key := convertReferenceToMetric(ctx, b.Config, r)
			if key != "" {
				used[key] = struct{}{}
			}
		}
		return nil
	})

	maxRefsLogged := 50

	// map iteration is randomized, which works for this case
	for key := range used {
		b.Metrics.AddBoolValue(key, true)
		maxRefsLogged--
		if maxRefsLogged <= 0 {
			break
		}
	}

	return nil
}

// convertReferenceToMetric converts a reference like "resources.jobs.foo.id" to
// a telemetry key like "resref__jobs__id"
func convertReferenceToMetric(ctx context.Context, cfg any, ref string) string {
	p, err := dyn.NewPathFromString(ref)
	if err != nil || len(p) < 3 || p[0].Key() != "resources" {
		return ""
	}

	group := truncate(p[1].Key(), maxFieldLength, "")
	kept := []string{"resref_" + group}

	for i := 3; i < len(p); i++ {
		c := p[i]
		if c.Key() != "" {
			item := c.Key()

			repl, err := censorValue(ctx, cfg, p[:i])
			if err != nil {
				kept[0] = "resreferr_" + group
				break
			}

			if repl != "" {
				item = repl
			}

			item = truncate(item, maxFieldLength, "")
			kept = append(kept, item)
			continue
		}
	}

	result := strings.Join(kept, ".")
	return truncate(result, maxMetricLength, "__cut")
}

func truncate(s string, n int, suffix string) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + suffix
}

func censorValue(ctx context.Context, v any, path dyn.Path) (string, error) {
	pathString := path.String()
	pathNode, err := structpath.Parse(pathString)
	if err != nil {
		log.Warnf(ctx, "internal error: parsing %q: %s", pathString, err)
		return "err", err
	}

	v, err = structaccess.Get(v, pathNode)
	if err != nil {
		log.Infof(ctx, "internal error: path=%s: %s", path, err)
		return "err", err
	}

	rv := reflect.ValueOf(v)
	for rv.IsValid() {
		switch rv.Kind() {
		case reflect.Pointer, reflect.Interface:
			if rv.IsNil() {
				return "", nil
			}
			rv = rv.Elem()
		default:
			if rv.Kind() == reflect.Map {
				return "*", nil
			}
			return "", nil
		}
	}
	return "", nil
}
