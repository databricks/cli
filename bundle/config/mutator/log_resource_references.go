package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// logResourceReferences scans resources for ${resources.*} references and logs them.
func LogResourceReferences() bundle.Mutator {
	return &logResourceReferences{}
}

type logResourceReferences struct{}

func (m *logResourceReferences) Name() string {
	return "LogResourceReferences"
}

func (m *logResourceReferences) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		pattern := dyn.NewPattern(dyn.Key("resources"))
		used := map[string]struct{}{}
		_, err := dyn.MapByPattern(root, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			_ = dyn.WalkReadOnly(v, func(path dyn.Path, val dyn.Value) error {
				ref, ok := dynvar.NewRef(val)
				if !ok {
					return nil
				}
				for _, r := range ref.References() {
					// Only track ${resources.*} references.
					if !strings.HasPrefix(r, "resources.") {
						continue
					}

					key := convertReferenceToMetric(r)
					if key != "" {
						used[key] = struct{}{}
					}
				}
				return nil
			})
			return v, nil
		})
		if err != nil {
			return dyn.InvalidValue, err
		}

		maxRefsLogged := 50

		// map iteration is randomized, which works for this case
		for key := range used {
			b.Metrics.AddBoolValue(key, true)
			maxRefsLogged--
			if maxRefsLogged <= 0 {
				break
			}
		}

		return root, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// convertReferenceToMetric converts a reference like "resources.jobs.foo.id" to
// a telemetry key like "jobs_id"; and deep paths like
// "resources.pipelines.bar.my.som.field[3].key.bla" to "pipelines_my_som_cut_4".
// It drops the resource key, keeps up to 2 key fields, truncates each to 15 chars,
// and appends _cut_N when there are remaining components (indices count too).
func convertReferenceToMetric(ref string) string {
	p, err := dyn.NewPathFromString(ref)
	if err != nil || len(p) < 3 || p[0].Key() != "resources" {
		return ""
	}

	kept := []string{"resref", truncate(p[1].Key(), 20)}
	remaining := 0
	for i := 3; i < len(p); i++ {
		c := p[i]
		if c.Key() != "" {
			if len(kept) < 4 {
				// Longest field name:
				// >>> len('hard_deletion_sync_min_interval_in_seconds') == 42
				kept = append(kept, truncate(c.Key(), 42))
				continue
			}
			remaining++
			continue
		}
		// index
		remaining++
	}

	if remaining > 0 {
		kept = append(kept, fmt.Sprintf("cut_%d", remaining))
	}
	return strings.Join(kept, "_")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
