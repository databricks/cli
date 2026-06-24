package fuzz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Difference is a single mismatch between the two engines' create payloads,
// located by a JSON-ish path (e.g. "tasks[0].new_cluster.num_workers").
type Difference struct {
	Path      string
	Direct    any
	Terraform any
}

func (d Difference) String() string {
	return fmt.Sprintf("%s: direct=%s terraform=%s", d.Path, render(d.Direct), render(d.Terraform))
}

// missing marks a value that is absent on one side.
type missing struct{}

func render(v any) string {
	if _, ok := v.(missing); ok {
		return "<absent>"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// DiffPayloads decodes both create payloads and returns every difference whose
// path is not explicitly ignored. ignorePaths are matched exactly against the
// rendered path, with "[*]" standing in for any slice index.
func DiffPayloads(direct, terraform json.RawMessage, ignorePaths []string) ([]Difference, error) {
	d, err := decode(direct)
	if err != nil {
		return nil, fmt.Errorf("decoding direct payload: %w", err)
	}
	tf, err := decode(terraform)
	if err != nil {
		return nil, fmt.Errorf("decoding terraform payload: %w", err)
	}

	var diffs []Difference
	diffValue("", d, tf, &diffs)

	ignore := make(map[string]bool, len(ignorePaths))
	for _, p := range ignorePaths {
		ignore[p] = true
	}

	filtered := diffs[:0]
	for _, diff := range diffs {
		if !ignore[normalizePath(diff.Path)] {
			filtered = append(filtered, diff)
		}
	}
	return filtered, nil
}

// decode unmarshals JSON using UseNumber so large int64 values (e.g. job ids,
// spark_context_id) are not corrupted by float64 rounding. See the encoding rule
// in the repo style guide.
func decode(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

func diffValue(path string, a, b any, diffs *[]Difference) {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			*diffs = append(*diffs, Difference{Path: path, Direct: a, Terraform: b})
			return
		}
		keys := unionKeys(av, bv)
		for _, k := range keys {
			achild, aok := av[k]
			bchild, bok := bv[k]
			child := joinKey(path, k)
			switch {
			case aok && bok:
				diffValue(child, achild, bchild, diffs)
			case aok:
				*diffs = append(*diffs, Difference{Path: child, Direct: achild, Terraform: missing{}})
			default:
				*diffs = append(*diffs, Difference{Path: child, Direct: missing{}, Terraform: bchild})
			}
		}
	case []any:
		bv, ok := b.([]any)
		if !ok {
			*diffs = append(*diffs, Difference{Path: path, Direct: a, Terraform: b})
			return
		}
		// Slices whose elements carry a natural identity key (tasks, job clusters)
		// are matched by that key so an engine emitting the same elements in a
		// different order is not reported as a difference. Everything else is
		// compared positionally.
		if key := identityKey(av, bv); key != "" {
			diffKeyedSlice(path, key, av, bv, diffs)
			return
		}
		n := max(len(av), len(bv))
		for i := range n {
			child := fmt.Sprintf("%s[%d]", path, i)
			switch {
			case i < len(av) && i < len(bv):
				diffValue(child, av[i], bv[i], diffs)
			case i < len(av):
				*diffs = append(*diffs, Difference{Path: child, Direct: av[i], Terraform: missing{}})
			default:
				*diffs = append(*diffs, Difference{Path: child, Direct: missing{}, Terraform: bv[i]})
			}
		}
	default:
		if !scalarEqual(a, b) {
			*diffs = append(*diffs, Difference{Path: path, Direct: a, Terraform: b})
		}
	}
}

// identityFields are the keys, in priority order, that uniquely identify the
// elements of a payload slice. Job tasks and shared job clusters are the slices
// whose order is not significant but which the engines may emit differently.
var identityFields = []string{"task_key", "job_cluster_key"}

// identityKey returns the field that identifies every element of both slices, or
// "" if the elements are not uniformly keyed objects (in which case the caller
// falls back to positional comparison).
func identityKey(a, b []any) string {
	for _, field := range identityFields {
		if allHaveKey(a, field) && allHaveKey(b, field) {
			return field
		}
	}
	return ""
}

func allHaveKey(s []any, field string) bool {
	if len(s) == 0 {
		return false
	}
	for _, el := range s {
		m, ok := el.(map[string]any)
		if !ok {
			return false
		}
		if _, ok := m[field].(string); !ok {
			return false
		}
	}
	return true
}

// diffKeyedSlice matches elements of a and b by the value of key (which is unique
// within each slice for tasks/job clusters) and diffs each matched pair,
// reporting unmatched elements as present-on-one-side. Paths keep numeric indices
// so ignore-path [*] normalization still applies.
func diffKeyedSlice(path, key string, a, b []any, diffs *[]Difference) {
	bByKey := make(map[string]any, len(b))
	for _, el := range b {
		bByKey[el.(map[string]any)[key].(string)] = el
	}

	matched := make(map[string]bool, len(a))
	for i, el := range a {
		child := fmt.Sprintf("%s[%d]", path, i)
		k := el.(map[string]any)[key].(string)
		matched[k] = true
		if bel, ok := bByKey[k]; ok {
			diffValue(child, el, bel, diffs)
		} else {
			*diffs = append(*diffs, Difference{Path: child, Direct: el, Terraform: missing{}})
		}
	}
	for j, el := range b {
		k := el.(map[string]any)[key].(string)
		if matched[k] {
			continue
		}
		child := fmt.Sprintf("%s[%d]", path, j)
		*diffs = append(*diffs, Difference{Path: child, Direct: missing{}, Terraform: el})
	}
}

// scalarEqual compares two JSON scalars. json.Number is compared by its string
// form so 1 and 1.0 don't masquerade as equal across engines.
func scalarEqual(a, b any) bool {
	an, aok := a.(json.Number)
	bn, bok := b.(json.Number)
	if aok && bok {
		return an.String() == bn.String()
	}
	return a == b
}

func unionKeys(a, b map[string]any) []string {
	seen := map[string]bool{}
	var keys []string
	for k := range a {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for k := range b {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}

func joinKey(path, key string) string {
	// Map keys can themselves contain dots or brackets (e.g. spark_conf entries
	// like "spark.databricks.delta.preview.enabled"). Render those as bracketed,
	// quoted segments so the path stays unambiguous and ignore entries can target
	// a single key.
	if key == "" || strings.ContainsAny(key, `.[]"`) {
		return path + "[" + strconv.Quote(key) + "]"
	}
	if path == "" {
		return key
	}
	return path + "." + key
}

// indexRe matches numeric slice indices like "[12]" but not quoted string keys
// like ["spark.x"].
var indexRe = regexp.MustCompile(`\[\d+\]`)

// normalizePath replaces concrete slice indices with [*] so a single ignore
// entry can cover every element of a slice.
func normalizePath(path string) string {
	return indexRe.ReplaceAllString(path, "[*]")
}

// DefaultIgnorePaths lists create-payload paths that legitimately differ between
// the engines and are not parity bugs. Keep this list small and well-justified;
// every entry is a known, intentional divergence.
var DefaultIgnorePaths = []string{
	// The terraform provider strips the deprecated/ignored spark conf
	// "spark.databricks.delta.preview.enabled" from new_cluster.spark_conf, while
	// the direct engine forwards it verbatim. The backend ignores the key either
	// way, so this is a benign provider-side filter rather than a parity bug.
	`tasks[*].new_cluster.spark_conf["spark.databricks.delta.preview.enabled"]`,
	`job_clusters[*].new_cluster.spark_conf["spark.databricks.delta.preview.enabled"]`,
}
