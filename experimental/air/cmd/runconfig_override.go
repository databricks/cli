package aircmd

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"go.yaml.in/yaml/v3"
)

// This file implements the `--override KEY=VALUE` flag. Overrides are applied to
// the parsed YAML map (not the typed runConfig) before re-decode, so one pipeline
// covers path existence, type coercion, and the semantic validate() rules.

// parseOverrides parses --override KEY=VALUE arguments, preserving order.
func parseOverrides(overrides []string) ([]overrideEntry, error) {
	entries := make([]overrideEntry, 0, len(overrides))
	for _, item := range overrides {
		key, value, found := strings.Cut(item, "=")
		if !found {
			// --override is repeatable, so a config path meant for -f can be
			// swallowed here; point at the real fix.
			hint := ""
			if strings.HasSuffix(item, ".yaml") || strings.HasSuffix(item, ".yml") {
				hint = fmt.Sprintf("; %q looks like a config file — pass it with -f/--file", item)
			}
			return nil, fmt.Errorf("invalid --override %q: expected KEY=VALUE (e.g. compute.num_accelerators=32)%s", item, hint)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid --override %q: empty key", item)
		}
		entries = append(entries, overrideEntry{path: key, raw: value})
	}
	return entries, nil
}

// overrideEntry is one parsed --override: its dotted path and the raw RHS string.
type overrideEntry struct {
	path string
	raw  string
}

// validateOverridePaths checks every dotted path against the runConfig schema
// before mutation, so an error names the exact --override key rather than the
// re-decode's Go-type language.
func validateOverridePaths(entries []overrideEntry) error {
	schema := configSchema()
	for _, e := range entries {
		if err := checkOverridePath(strings.Split(e.path, "."), schema, e.path); err != nil {
			return err
		}
	}
	return nil
}

// checkOverridePath validates one dotted path against a resolved schema node.
// It shares configSchema()'s reflection walk with `-h config.<field>` but keeps
// the --override error voice, which names the offending flag.
func checkOverridePath(parts []string, node configField, fullPath string) error {
	name := parts[0]
	child, ok := findConfigChild(node.children, name)
	if !ok {
		names := make([]string, 0, len(node.children))
		for _, c := range node.children {
			names = append(names, configLeafName(c.path))
		}
		slices.Sort(names)
		return fmt.Errorf("invalid --override %q: %q is not a known field; available fields are: %s",
			fullPath, name, strings.Join(names, ", "))
	}
	if len(parts) == 1 {
		return nil
	}
	// A free-form map's keys are user-defined, so any sub-path into it is valid.
	if child.freeForm {
		return nil
	}
	if len(child.children) == 0 {
		return fmt.Errorf("invalid --override %q: %q is not a nested object; cannot address sub-field %q",
			fullPath, name, strings.Join(parts[1:], "."))
	}
	return checkOverridePath(parts[1:], child, fullPath)
}

// applyOverrides walks each dotted path into the parsed YAML map and sets the
// leaf to the RHS parsed as a YAML scalar. Intermediate maps are auto-created so
// an override can add a field the YAML omits; the later re-decode rejects paths
// absent from the schema. Changes are logged to stderr to keep JSON stdout clean.
func applyOverrides(ctx context.Context, m map[string]any, entries []overrideEntry) error {
	for _, e := range entries {
		var value any
		if err := yaml.Unmarshal([]byte(e.raw), &value); err != nil {
			return fmt.Errorf("invalid --override %q: cannot parse value %q: %w", e.path, e.raw, err)
		}

		parts := strings.Split(e.path, ".")
		current := m
		for _, part := range parts[:len(parts)-1] {
			next, ok := current[part].(map[string]any)
			if !ok {
				next = map[string]any{}
				current[part] = next
			}
			current = next
		}

		leaf := parts[len(parts)-1]
		old, had := current[leaf]
		current[leaf] = value
		if had {
			logOverride(ctx, fmt.Sprintf("Override: changing %s from %v to %v", e.path, old, value))
		} else {
			logOverride(ctx, fmt.Sprintf("Override: setting %s to %v", e.path, value))
		}
	}
	return nil
}

// logOverride writes to stderr only when a cmdIO is present; cmdio.LogString
// panics without one, as in non-command callers such as unit tests.
func logOverride(ctx context.Context, msg string) {
	if cmdio.HasIO(ctx) {
		cmdio.LogString(ctx, msg)
	}
}
