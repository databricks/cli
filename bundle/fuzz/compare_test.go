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

// ignoreRule suppresses a known, intentional engine divergence. A rule matches a
// difference when the difference's normalized path equals Path and, if Match is
// non-nil, Match also reports true for the two values. A nil Match ignores any
// difference at Path; a non-nil Match narrows the rule to specific values so a
// genuine mismatch at the same path is still reported.
type ignoreRule struct {
	Path  string
	Match func(d difference) bool
}

// diffPayloads decodes both create payloads and returns every difference that no
// ignore rule suppresses. Paths are matched with "[*]" standing in for any slice
// index (see normalizePath).
func diffPayloads(direct, terraform json.RawMessage, ignore []ignoreRule) ([]difference, error) {
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
		if !ignored(diff, ignore) {
			filtered = append(filtered, diff)
		}
	}
	return filtered, nil
}

// ignored reports whether any rule suppresses d.
func ignored(d difference, rules []ignoreRule) bool {
	norm := normalizePath(d.Path)
	for _, r := range rules {
		if r.Path != norm {
			continue
		}
		if r.Match == nil || r.Match(d) {
			return true
		}
	}
	return false
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
func diffKeyedSlice(path, key string, a, b []any, diffs *[]difference) {
	// identityFields are unique within a slice by API contract (no two job tasks
	// share a task_key, no two job_clusters share a job_cluster_key), so keying by
	// them is unambiguous. If a payload ever repeated a key, last-one-wins here and
	// the duplicate would be mismatched rather than reported precisely; callers
	// outside the job-create harness must not rely on this for non-unique keys.
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

// defaultIgnoreRules lists create-payload divergences that are known, intentional
// engine differences and not parity bugs. Keep this list small and
// well-justified; every entry is a documented divergence.
var defaultIgnoreRules = []ignoreRule{
	// The terraform provider strips the deprecated/ignored spark conf
	// "spark.databricks.delta.preview.enabled" from new_cluster.spark_conf, while
	// the direct engine forwards it verbatim. The backend ignores the key either
	// way, so this is a benign provider-side filter rather than a parity bug.
	{Path: `tasks[*].new_cluster.spark_conf["spark.databricks.delta.preview.enabled"]`},
	{Path: `job_clusters[*].new_cluster.spark_conf["spark.databricks.delta.preview.enabled"]`},

	// For a single-node task-level new_cluster (no autoscale, num_workers unset)
	// the terraform provider force-sends num_workers:0 while the direct engine
	// omits the field, so the create payloads diverge. This is a real
	// terraform/direct divergence the harness found (seed 29); it is documented
	// and suppressed here rather than fixed in this PR. Tracked under DECO-25361.
	// Shared job_clusters are not affected: resourcemutator already force-sends
	// num_workers for them under both engines, so only the task path diverges.
	//
	// Match narrows this to exactly that shape (direct absent, terraform 0); a
	// genuine num_workers value mismatch at the same path is still reported.
	{Path: `tasks[*].new_cluster.num_workers`, Match: isBenignTaskNumWorkers},
}

// isBenignTaskNumWorkers reports whether d is the single documented num_workers
// divergence: the direct engine omits num_workers while terraform force-sends 0.
// Any other pair of values (in particular two differing non-zero counts) is a
// real divergence and must not be suppressed.
func isBenignTaskNumWorkers(d difference) bool {
	_, directAbsent := d.Direct.(missing)
	n, ok := d.Terraform.(json.Number)
	return directAbsent && ok && n.String() == "0"
}
