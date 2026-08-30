package tests

import (
	"bytes"
	"errors"
	"fmt"
	"hash/fnv"
	"io/fs"
	"maps"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// absent is the "field not present in the config" value. Every transition is a move
// between two values drawn from a field's set, and add/remove are just the moves
// with absent on one side.
var absent = dyn.InvalidValue

// fieldValues is the per-resource-type value library, e.g. testdata/fields/schemas.yml.
//
// A field with no entry falls back to defaultValues for its Go kind. Fields the backend
// constrains (enums, ids, cross-referenced names) need an explicit entry, otherwise
// every value is rejected and the field reports only BACKEND_ERROR.
type fieldValues struct {
	// skip lists field paths to leave out entirely, with a reason. A key ending in ".*"
	// skips that whole subtree.
	skip map[string]string

	// fields maps a field path to the values to try.
	fields map[string][]dyn.Value

	// base is merged into the resource before the first deploy. It makes a block
	// reachable that the invariant config does not declare: a coherent git_source, say,
	// whose fields cannot be tested one at a time from nothing because the API only
	// accepts the set as a whole. Once seeded, each of its fields varies normally.
	//
	// Keep path-valued fields out of it: it is merged after the mutator pipeline has run,
	// so nothing here goes through path translation.
	base dyn.Value
}

// fieldsDir holds the per-resource-type value libraries.
const fieldsDir = "testdata/fields"

// declaredUnsettable collects the fields the resource declares as backend outputs. A user
// cannot meaningfully set one at all -- they are in the input schema by accident -- so they
// are skipped and the resource's own reason recorded instead of spending six deploys per
// field rediscovering it.
//
// This and declaredIgnoredLocally are the only places the suite reads resources.yml.
func declaredUnsettable(adapter *dresources.Adapter) []dresources.FieldRule {
	var rules []dresources.FieldRule
	for _, cfg := range lifecycleConfigs(adapter) {
		// Only outputs are unsettable. An input_only or managed field is one the backend
		// owns on read but the user may still legitimately write, so those stay in.
		for _, rule := range cfg.IgnoreRemoteChanges {
			if strings.HasSuffix(rule.Reason, "output_only") {
				rules = append(rules, rule)
			}
		}
	}
	return rules
}

// declaredIgnoredLocally collects the fields a resource says it drops local changes to.
// These are *not* skipped: the declaration is a claim about behaviour, and the suite is in a
// position to check it. A transition of one should come back suppressed with that same
// reason; anything else means the field is not actually inert and the declaration is wrong.
func declaredIgnoredLocally(adapter *dresources.Adapter) []dresources.FieldRule {
	var rules []dresources.FieldRule
	for _, cfg := range lifecycleConfigs(adapter) {
		rules = append(rules, cfg.IgnoreLocalChanges...)
	}
	return rules
}

func lifecycleConfigs(adapter *dresources.Adapter) []*dresources.ResourceLifecycleConfig {
	var out []*dresources.ResourceLifecycleConfig
	for _, cfg := range []*dresources.ResourceLifecycleConfig{adapter.ResourceConfig(), adapter.GeneratedResourceConfig()} {
		if cfg != nil {
			out = append(out, cfg)
		}
	}
	return out
}

// ruleReason reports the reason a field matches one of the given rules. Accepts both a
// concrete path ("tags[0].key") and a pattern ("tags[*].key"): the second form is needed
// for a field whose container the config does not declare, where no concrete path exists.
func ruleReason(rules []dresources.FieldRule, path string) (string, bool) {
	if concrete, err := structpath.ParsePath(path); err == nil {
		for _, rule := range rules {
			// The same match the planner makes, so the suite and the engine agree on scope.
			if concrete.HasPatternPrefix(rule.Field) {
				return rule.Reason, true
			}
		}
		return "", false
	}

	// A pattern: compare textually, since one pattern cannot be matched against another.
	for _, rule := range rules {
		declared := rule.Field.String()
		if path == declared || strings.HasPrefix(path, declared+".") || strings.HasPrefix(path, declared+"[") {
			return rule.Reason, true
		}
	}
	return "", false
}

// cliManagedFields are fields the bundle acts on itself and never sends to any API, so no
// transition of one can appear in a plan. Only fields with no API counterpart at all belong
// here: a field the CLI merely *overwrites* is still worth testing, because the resulting
// verdict is the evidence that it does.
var cliManagedFields = map[string]string{
	"lifecycle.prevent_destroy": "acted on by the bundle, never sent to any API; gates destroy",
}

// loadFieldValues reads testdata/fields/<resource_type>.yml. Parsing goes through the repo's own
// yamlloader rather than a yaml package, so the values arrive as dyn.Value -- the same
// representation the bundle config uses, which is what they are written into.
func loadFieldValues(resourceType string) (*fieldValues, error) {
	fv := &fieldValues{skip: map[string]string{}, fields: map[string][]dyn.Value{}, base: dyn.InvalidValue}
	maps.Copy(fv.skip, cliManagedFields)

	path := filepath.Join(fieldsDir, resourceType+".yml")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fv, nil
	}
	if err != nil {
		return nil, err
	}

	root, err := yamlloader.LoadYAML(path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	if v, err := dyn.Get(root, "skip"); err == nil {
		m, _ := v.AsMap()
		for _, pair := range m.Pairs() {
			key, _ := pair.Key.AsString()
			reason, _ := pair.Value.AsString()
			fv.skip[key] = reason
		}
	}
	if v, err := dyn.Get(root, "base"); err == nil {
		fv.base = v
	}
	if v, err := dyn.Get(root, "fields"); err == nil {
		m, _ := v.AsMap()
		for _, pair := range m.Pairs() {
			key, _ := pair.Key.AsString()
			values, ok := pair.Value.AsSequence()
			if !ok {
				return nil, fmt.Errorf("%s: fields.%s must be a list", path, key)
			}
			fv.fields[key] = values
		}
	}

	return fv, nil
}

// isSubResource reports whether a path belongs to a sub-resource block.
func isSubResource(path string) bool {
	name, _, _ := strings.Cut(path, ".")
	name, _, _ = strings.Cut(name, "[")
	return slices.Contains(subResourceKinds, name)
}

// isContainer reports whether a kind has fields or elements to descend into.
func isContainer(kind reflect.Kind) bool {
	switch kind {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}

// defaultValues per Go kind. Two values per kind is enough to observe a value->value
// transition on top of add and remove; more would multiply the matrix without testing a
// different code path.
//
//nolint:exhaustive // the default branch covers every kind without a generic value
func defaultValues(kind reflect.Kind) []dyn.Value {
	switch kind {
	case reflect.Bool:
		return []dyn.Value{dyn.V(false), dyn.V(true)}
	case reflect.String:
		return []dyn.Value{dyn.V("x"), dyn.V("y")}
	case reflect.Int, reflect.Int32, reflect.Int64:
		return []dyn.Value{dyn.V(1), dyn.V(2)}
	case reflect.Float32, reflect.Float64:
		return []dyn.Value{dyn.V(1.0), dyn.V(2.0)}
	default:
		// The remaining kinds (interface, unsigned, complex, ...) have no meaningful
		// generic value; such a field needs an entry in the value library.
		return nil
	}
}

// field is one testable leaf of the resource's input struct.
type field struct {
	// seed drives this field's value order and the order its transitions are walked in.
	// Derived from the run seed and the field path, so it is stable for a given commit
	// and different for every field.
	seed uint64

	path   string // structpath/dyn path, e.g. "comment" or "email_notifications.on_failure"
	kind   reflect.Kind
	values []dyn.Value

	// required fields do not get an "absent" transition: a config missing one is
	// rejected by bundle validate, so removing it is not something a user can deploy.
	required bool
}

// transition is one move of a field from one value to another.
type transition struct {
	from, to dyn.Value
}

// label names the transition for the subtest. Kept free of shell metacharacters so a
// single case can be re-run without quoting:
//
//	go test ./bundle/direct/tests -run TestFields/pipelines/.*/dry_run/absent_to_true -v
func (t transition) label() string {
	return valueLabel(t.from) + "_to_" + valueLabel(t.to)
}

// transitions returns every ordered pair of the field's values, ordered as a single walk:
// each transition starts where the previous one ended, so reaching a starting value costs
// nothing and the field is deployed once per transition instead of twice. absent is one of
// the values, so adding and removing the field are the pairs with absent on one side.
//
// The values form a complete digraph, where every vertex has equal in- and out-degree, so
// an Eulerian circuit always exists and covers each ordered pair exactly once. The order
// among a vertex's outgoing edges is shuffled from a seed derived from the field path:
// fixed for a given field, so the report stays byte-stable, but different between fields,
// so the suite is not always walking the same shape of path.
func (f field) transitions() []transition {
	values := f.values
	if !f.required {
		values = append([]dyn.Value{absent}, values...)
	}
	if len(values) < 2 {
		return nil
	}

	rng := rand.New(rand.NewPCG(f.seed, 0))
	rng.Shuffle(len(values), func(i, j int) { values[i], values[j] = values[j], values[i] })
	unused := make([][]int, len(values))
	for from := range values {
		for to := range values {
			if from != to {
				unused[from] = append(unused[from], to)
			}
		}
		rng.Shuffle(len(unused[from]), func(i, j int) {
			unused[from][i], unused[from][j] = unused[from][j], unused[from][i]
		})
	}

	// Hierholzer: walk until stuck, then splice in circuits from vertices that still have
	// unused edges. The reversed visit order is the circuit.
	var stack, circuit []int
	stack = append(stack, 0)
	for len(stack) > 0 {
		v := stack[len(stack)-1]
		if len(unused[v]) == 0 {
			circuit = append(circuit, v)
			stack = stack[:len(stack)-1]
			continue
		}
		next := unused[v][0]
		unused[v] = unused[v][1:]
		stack = append(stack, next)
	}
	slices.Reverse(circuit)

	out := make([]transition, 0, len(circuit)-1)
	for i := 1; i < len(circuit); i++ {
		out = append(out, transition{from: values[circuit[i-1]], to: values[circuit[i]]})
	}
	return out
}

// fieldSeed mixes the run seed with a field path, so every field gets its own order while
// the whole run stays reproducible from the commit alone.
func fieldSeed(runSeed uint64, path string) uint64 {
	h := fnv.New64a()
	_, _ = fmt.Fprintf(h, "%d/%s", runSeed, path)
	return h.Sum64()
}

// requiredFields returns the field names the bundle schema marks required directly
// under the given path inside a resource, e.g. "" for the resource itself or
// "evaluation" for a nested object. The data is generated from the config structs
// (bundle/internal/validation), not from the engine's resources.yml.
func requiredFields(resourceType, parent string) []string {
	key := "resources." + resourceType + ".*"
	if parent != "" {
		key += "." + parent
	}
	return generated.RequiredFields[key]
}

// skipReason returns why a field is excluded, if it is. A "<prefix>.*" key covers every
// field beneath it, for a block that only works as a whole.
func (fv *fieldValues) skipReason(path string) (string, bool) {
	if reason, ok := fv.skip[path]; ok {
		return reason, true
	}
	for key, reason := range fv.skip {
		if prefix, ok := strings.CutSuffix(key, ".*"); ok && strings.HasPrefix(path, prefix+".") {
			return reason, true
		}
	}
	return "", false
}

func isRequired(resourceType, path string) bool {
	parent, name := "", path
	if i := strings.LastIndex(path, "."); i >= 0 {
		parent, name = path[:i], path[i+1:]
	}
	return slices.Contains(requiredFields(resourceType, parent), name)
}

// enumerateFields walks the resource's input config type the way cmd/bundle/debug
// refschema does, and pairs each field with the values to try.
//
// base is the resource as the config actually declares it, which is what makes slices and
// maps testable. A slice or map the config populates becomes a field in its own right,
// with add-an-entry and remove-an-entry transitions; and a pattern like
// "tasks[*].description" is expanded to the indices that exist, so the fields inside an
// element get the same treatment as any other. A pattern with nothing behind it in the
// config is reported as not covered rather than silently tested against nothing.
func enumerateFields(resourceType string, inputType reflect.Type, fv *fieldValues, base dyn.Value, runSeed uint64, unsettable []dresources.FieldRule) (fields []field, wildcard []string, inertFields map[string]string) {
	inertFields = map[string]string{}
	add := func(path string, kind reflect.Kind, values []dyn.Value) {
		if values == nil {
			return
		}
		if _, skipped := fv.skipReason(path); skipped {
			return
		}
		if reason, ok := ruleReason(unsettable, path); ok {
			inertFields[path] = reason
			return
		}
		fields = append(fields, field{
			seed:     fieldSeed(runSeed, path),
			path:     path,
			kind:     kind,
			values:   values,
			required: isRequired(resourceType, path),
		})
	}

	_ = structwalk.WalkType(inputType, func(p *structpath.PatternNode, typ reflect.Type, sf *reflect.StructField) bool {
		if p.IsRoot() {
			return true
		}
		for typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if sf != nil && isNotUserSettable(sf) {
			return false
		}

		path := p.String()
		if isSubResource(path) {
			// permissions and grants are separate plan nodes with their own adapters, and
			// this suite strips them from every config. They are out of scope as subjects,
			// so they are not fields here and not gaps either.
			return false
		}
		isWildcard := strings.Contains(path, "*")

		if isContainer(typ.Kind()) {
			// A struct is only a grouping; a slice or map is also a value the user can
			// grow and shrink, so test it as a field before descending into it.
			if typ.Kind() != reflect.Struct {
				for _, concrete := range expandPattern(base, path) {
					add(concrete, typ.Kind(), containerValues(base, concrete))
				}
			}
			return true
		}

		if !isWildcard {
			values := fv.fields[path]
			if values == nil {
				values = defaultValues(typ.Kind())
			}
			add(path, typ.Kind(), values)
			return false
		}

		concrete := expandPattern(base, path)
		if len(concrete) == 0 {
			// A field the user cannot set is not a coverage gap. Checked on the pattern here
			// because there is no concrete path to check: the container is absent.
			if reason, ok := ruleReason(unsettable, path); ok {
				inertFields[path] = reason
				return false
			}
			if _, skipped := fv.skipReason(path); !skipped {
				wildcard = append(wildcard, path)
			}
			return false
		}
		for _, c := range concrete {
			// Explicit values are keyed by the pattern, since an index is incidental.
			values := fv.fields[path]
			if values == nil {
				values = defaultValues(typ.Kind())
			}
			add(c, typ.Kind(), values)
		}
		return false
	})

	// Shuffle the fields too. Each field restores the config before the next one starts,
	// so the order costs nothing in requests -- but a field that only passes because of
	// what ran before it should not keep getting away with it.
	rng := rand.New(rand.NewPCG(runSeed, 1))
	rng.Shuffle(len(fields), func(i, j int) { fields[i], fields[j] = fields[j], fields[i] })

	return fields, wildcard, inertFields
}

// isNotUserSettable reports whether the bundle marks a field as something the user
// never writes: an output the CLI fills in (id, url) or an internal bookkeeping field.
func isNotUserSettable(sf *reflect.StructField) bool {
	tag := sf.Tag.Get("bundle")
	return strings.Contains(tag, "readonly") ||
		strings.Contains(tag, "internal") ||
		sf.Tag.Get("json") == "-"
}

// containerValues returns the values to try for a slice or map field that the base config
// actually populates: the config's own value, and that value with its last entry dropped.
// Combined with the implicit "absent", one field then covers adding and removing the whole
// container as well as adding and removing a single entry -- all with data the backend has
// already accepted, so nothing has to be invented.
func containerValues(base dyn.Value, path string) []dyn.Value {
	current, err := dyn.Get(base, path)
	if err != nil || !current.IsValid() {
		return nil
	}

	switch current.Kind() {
	case dyn.KindSequence:
		items, _ := current.AsSequence()
		if len(items) == 0 {
			return nil
		}
		return []dyn.Value{current, dyn.V(slices.Clone(items[:len(items)-1]))}

	case dyn.KindMap:
		m, _ := current.AsMap()
		pairs := m.Pairs()
		if len(pairs) == 0 {
			return nil
		}
		trimmed := dyn.NewMapping()
		for _, pair := range pairs[:len(pairs)-1] {
			key, _ := pair.Key.AsString()
			trimmed.SetLoc(key, pair.Key.Locations(), pair.Value)
		}
		return []dyn.Value{current, dyn.NewValue(trimmed, current.Locations())}

	default:
		return nil
	}
}

// expandPattern turns a pattern from the type walk into the concrete paths the base config
// actually has: "tasks[*].description" against a config with one task yields
// "tasks[0].description". A pattern with nothing behind it yields nothing, which is how a
// field under an absent container is reported as not covered.
func expandPattern(base dyn.Value, pattern string) []string {
	paths := []string{""}
	for _, seg := range splitPattern(pattern) {
		var next []string
		for _, prefix := range paths {
			switch seg {
			case "[*]":
				value, err := dyn.Get(base, prefix)
				if err != nil {
					continue
				}
				items, ok := value.AsSequence()
				if !ok {
					continue
				}
				for i := range items {
					next = append(next, prefix+"["+strconv.Itoa(i)+"]")
				}
			case "*":
				value, err := dyn.Get(base, prefix)
				if err != nil {
					continue
				}
				m, ok := value.AsMap()
				if !ok {
					continue
				}
				for _, pair := range m.Pairs() {
					key, _ := pair.Key.AsString()
					next = append(next, prefix+"["+quoteKey(key)+"]")
				}
			default:
				next = append(next, joinPath(prefix, seg))
			}
		}
		paths = next
	}
	return paths
}

// splitPattern breaks a pattern into field names, "[*]" and "*" segments.
func splitPattern(pattern string) []string {
	var segs []string
	for part := range strings.SplitSeq(pattern, ".") {
		// A dot-separated part is either the map wildcard on its own, or a field name
		// followed by any number of [*] element wildcards.
		if part == "*" {
			segs = append(segs, "*")
			continue
		}
		for {
			name, rest, found := strings.Cut(part, "[*]")
			if name != "" {
				segs = append(segs, name)
			}
			if !found {
				break
			}
			segs = append(segs, "[*]")
			part = rest
		}
	}
	return segs
}

func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// quoteKey renders a map key the way structpath and dyn both parse it back.
func quoteKey(key string) string {
	return "'" + strings.ReplaceAll(key, "'", "''") + "'"
}
