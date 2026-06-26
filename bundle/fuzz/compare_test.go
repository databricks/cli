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

// difference is a single mismatch between the two engines' create payloads,
// located by a JSON-ish path (e.g. "tasks[0].new_cluster.num_workers").
type difference struct {
	Path      string
	Direct    any
	Terraform any
}

func (d difference) String() string {
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

// diffPayloads decodes both create payloads and returns every difference whose
// normalized path is not in ignore ("[*]" stands in for any slice index, see
// normalizePath).
func diffPayloads(direct, terraform json.RawMessage, ignore []string) ([]difference, error) {
	d, err := decode(direct)
	if err != nil {
		return nil, fmt.Errorf("decoding direct payload: %w", err)
	}
	tf, err := decode(terraform)
	if err != nil {
		return nil, fmt.Errorf("decoding terraform payload: %w", err)
	}

	var diffs []difference
	diffValue("", d, tf, &diffs)

	filtered := diffs[:0]
	for _, diff := range diffs {
		if !slices.Contains(ignore, normalizePath(diff.Path)) {
			filtered = append(filtered, diff)
		}
	}
	return filtered, nil
}

// decode unmarshals JSON with UseNumber so large int64 values (job ids,
// spark_context_id) aren't corrupted by float64 rounding.
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

func diffValue(path string, a, b any, diffs *[]difference) {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			*diffs = append(*diffs, difference{Path: path, Direct: a, Terraform: b})
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
				*diffs = append(*diffs, difference{Path: child, Direct: achild, Terraform: missing{}})
			default:
				*diffs = append(*diffs, difference{Path: child, Direct: missing{}, Terraform: bchild})
			}
		}
	case []any:
		bv, ok := b.([]any)
		if !ok {
			*diffs = append(*diffs, difference{Path: path, Direct: a, Terraform: b})
			return
		}
		// Match keyed slices (tasks, job clusters) by identity so a different emit
		// order isn't a difference; everything else is compared positionally.
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
				*diffs = append(*diffs, difference{Path: child, Direct: av[i], Terraform: missing{}})
			default:
				*diffs = append(*diffs, difference{Path: child, Direct: missing{}, Terraform: bv[i]})
			}
		}
	default:
		if !scalarEqual(a, b) {
			*diffs = append(*diffs, difference{Path: path, Direct: a, Terraform: b})
		}
	}
}

// identityFields are the keys, in priority order, that uniquely identify the
// elements of order-insensitive payload slices (job tasks, shared job clusters).
var identityFields = []string{"task_key", "job_cluster_key"}

// identityKey returns the field that identifies every element of both slices, or
// "" if they are not uniformly keyed objects (caller then compares positionally).
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

// diffKeyedSlice matches elements of a and b by key (unique within each slice for
// tasks/job clusters by API contract) and diffs each matched pair, reporting
// unmatched elements as present-on-one-side. Paths keep numeric indices so [*]
// normalization still applies. Duplicate keys would be last-one-wins.
func diffKeyedSlice(path, key string, a, b []any, diffs *[]difference) {
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
			*diffs = append(*diffs, difference{Path: child, Direct: el, Terraform: missing{}})
		}
	}
	for j, el := range b {
		k := el.(map[string]any)[key].(string)
		if matched[k] {
			continue
		}
		child := fmt.Sprintf("%s[%d]", path, j)
		*diffs = append(*diffs, difference{Path: child, Direct: missing{}, Terraform: el})
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
	// Map keys can contain dots/brackets (e.g. spark_conf keys), so render those as
	// bracketed quoted segments to keep the path unambiguous.
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

// defaultIgnorePaths lists known, intentional engine divergences. Keep it small;
// every entry is a documented difference, not a parity bug.
var defaultIgnorePaths = []string{
	// Terraform strips the deprecated "spark.databricks.delta.preview.enabled" from
	// spark_conf while direct forwards it. The backend ignores it either way.
	`tasks[*].new_cluster.spark_conf["spark.databricks.delta.preview.enabled"]`,
	`job_clusters[*].new_cluster.spark_conf["spark.databricks.delta.preview.enabled"]`,
}
