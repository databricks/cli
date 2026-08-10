package dresources

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
)

// RedactSensitiveConfigValues walks the bundle config dyn.Value and replaces the
// value of every field declared as sensitive_fields for its resource type with
// "[redacted]". This is used before printing the full config to stdout (e.g.
// `bundle validate -o json`) so plaintext secrets are never shown to the user.
//
// The function only handles resource types that have at least one sensitive_field
// declared in resources.yml. Each sensitive field pattern is translated to a
// dyn.Pattern of the form: resources.<type>.*.<field_path>.
func RedactSensitiveConfigValues(v dyn.Value) (dyn.Value, error) {
	patterns := buildSensitivePatterns()
	for _, pat := range patterns {
		var err error
		v, err = dyn.MapByPattern(v, pat, func(p dyn.Path, _ dyn.Value) (dyn.Value, error) {
			return dyn.V(sensitiveRedactedMarker), nil
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}
	return v, nil
}

const sensitiveRedactedMarker = "[redacted]"

// buildSensitivePatterns returns one dyn.Pattern per sensitive field rule across
// all resource types. Each pattern covers: resources.<type>.*.<field_path>.
func buildSensitivePatterns() []dyn.Pattern {
	cfg := MustLoadConfig()
	var patterns []dyn.Pattern
	for resourceType, rc := range cfg.Resources {
		for _, rule := range rc.SensitiveFields {
			if rule.Field == nil {
				continue
			}
			fieldPat := structPathToDynPattern(rule.Field)
			if fieldPat == nil {
				continue
			}
			// resources.<type>.* + field pattern components
			base := dyn.NewPattern(
				dyn.Key("resources"),
				dyn.Key(resourceType),
				dyn.AnyKey(),
			)
			full := append(base, fieldPat...)
			patterns = append(patterns, full)
		}
	}
	return patterns
}

// structPathToDynPattern converts a structpath.PatternNode to a slice of
// dyn.patternComponent values. Returns nil if conversion is not possible.
func structPathToDynPattern(node *structpath.PatternNode) dyn.Pattern {
	if node == nil || node.IsRoot() {
		return nil
	}

	segments := node.AsSlice()
	pat := make(dyn.Pattern, 0, len(segments))
	for _, seg := range segments {
		if seg.BracketStar() || seg.DotStar() {
			pat = append(pat, dyn.AnyKey())
		} else if idx, ok := seg.Index(); ok {
			pat = append(pat, dyn.Index(idx))
		} else if key, ok := seg.StringKey(); ok {
			pat = append(pat, dyn.Key(key))
		} else {
			// Unsupported segment type; skip entire pattern.
			return nil
		}
	}
	return pat
}
