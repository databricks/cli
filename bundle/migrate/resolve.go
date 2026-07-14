package migrate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
)

// evaluateTemplate evaluates a template string like "${resources.pipelines.bar.cluster[0].label}"
// by looking up each ${...} reference from TF state.
func evaluateTemplate(state TFStateAttrs, template string) (string, error) {
	ref, ok := dynvar.NewRef(dyn.V(template))
	if !ok {
		return template, nil
	}

	result := template
	for _, pathString := range ref.References() {
		path, err := structpath.ParsePath(pathString)
		if err != nil {
			return "", fmt.Errorf("cannot parse reference path %q: %w", pathString, err)
		}
		// Expect resources.<group>.<name>.<field...>
		if path.Len() < 4 {
			return "", fmt.Errorf("unexpected reference format (too short): %q", pathString)
		}
		// Check first component is "resources"
		firstNode := path.Prefix(1)
		if firstNode.String() != "resources" {
			return "", fmt.Errorf("unexpected reference format (expected resources.*): %q", pathString)
		}

		group := path.SkipPrefix(1).Prefix(1).String()
		name := path.SkipPrefix(2).Prefix(1).String()
		fieldPath := path.SkipPrefix(3)

		value, err := LookupTFField(state, group, name, fieldPath)
		if err != nil {
			return "", fmt.Errorf("cannot look up %q: %w", pathString, err)
		}

		result = strings.ReplaceAll(result, "${"+pathString+"}", fmt.Sprintf("%v", value))
	}
	return result, nil
}

// ResolveFieldRef resolves a single reference for a field in resource (srcGroup, srcName).
// fieldPath is the path of the field within the source resource (in DABs naming, from sv.Refs key).
// refTemplate is the template string for that field, e.g. "${resources.pipelines.bar.cluster[0].label}".
//
// Two methods are tried:
//   - Method A: read the field from the source resource's own TF state.
//   - Method B: evaluate the template by reading each referenced field from TF state.
//
// Returns the reconciled value or an error if both methods fail. The bool return
// reports whether a warning was logged (methods disagreed); warnPrefix is
// prepended to that warning so background callers (the post-deploy dry-run) can
// attribute it.
func ResolveFieldRef(ctx context.Context, state TFStateAttrs, srcGroup, srcName string, fieldPath *structpath.PathNode, refTemplate, warnPrefix string) (any, bool, error) {
	// Method A: read field from source resource's TF state.
	valueA, errA := LookupTFField(state, srcGroup, srcName, fieldPath)

	// Method B: evaluate the template by looking up each reference.
	valueB, errB := evaluateTemplate(state, refTemplate)

	switch {
	case errA == nil && errB == nil:
		aStr := fmt.Sprintf("%v", valueA)
		if aStr == valueB {
			return valueA, false, nil
		}
		// Both succeeded but disagree: prefer longer string and warn.
		if len(valueB) > len(aStr) {
			log.Warnf(ctx, warnPrefix+"resource %s.%s field %s: method A value %q and method B value %q disagree; using longer (method B)",
				srcGroup, srcName, fieldPath, aStr, valueB)
			return valueB, true, nil
		}
		log.Warnf(ctx, warnPrefix+"resource %s.%s field %s: method A value %q and method B value %q disagree; using longer (method A)",
			srcGroup, srcName, fieldPath, aStr, valueB)
		return valueA, true, nil
	case errA == nil:
		return valueA, false, nil
	case errB == nil:
		return valueB, false, nil
	default:
		return nil, false, fmt.Errorf("%s.%s field %s: method A: %w; method B: %w",
			srcGroup, srcName, fieldPath, errA, errB)
	}
}
