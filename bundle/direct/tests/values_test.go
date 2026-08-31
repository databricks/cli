package tests

import (
	"encoding/json"
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
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"go.yaml.in/yaml/v3"
)

// absent is the "field not present in the config" value. Every transition is a move
// between two values drawn from a field's set, and add/remove are just the moves
// with absent on one side. A Go struct has no absent, so nil stands for it: writing it
// means the zero value with the field dropped from ForceSendFields.
var absent any = nil

// fieldValues is the per-resource-type value library, e.g. testdata/fields/schemas.yml.
//
// A field with no entry falls back to defaultValues for its Go kind. Fields the backend
// constrains (enums, ids, cross-referenced names) need an explicit entry, otherwise
// every value is rejected and the field reports only BACKEND_ERROR.
type fieldValues struct {
	// skip lists field paths to leave out entirely, with a reason. A key ending in ".*"
	// skips that whole subtree.
	skip map[string]string

	// fields maps a field path to the values to try, as the YAML gave them. They are
	// converted to the field's own Go type when the field is enumerated.
	fields map[string][]any

	// base is merged into the resource before the first deploy. It makes a block
	// reachable that the invariant config does not declare: a coherent git_source, say,
	// whose fields cannot be tested one at a time from nothing because the API only
	// accepts the set as a whole. Once seeded, each of its fields varies normally.
	//
	// Keep path-valued fields out of it: it is merged after the mutator pipeline has run,
	// so nothing here goes through path translation.
	//
	base any

	// localOnly is the reason this resource type cannot be driven against a real workspace:
	// it needs workspace or cloud state the suite does not provision (a storage credential
	// with IAM behind it), or the service is not available in every cloud. Mirrors the
	// per-config cloud exclusions in acceptance/bundle/invariant/test.toml.
	//
	// Such a type is skipped on cloud rather than reported: the local golden stands, and a
	// cloud run neither confirms nor contradicts it.
	localOnly string
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

// loadFieldValues reads testdata/fields/<resource_type>.yml, expanding the same $VARS the
// corpus configs use so a value can name the workspace's own user rather than a placeholder
// only the fake server knows.
func loadFieldValues(resourceType string, vars map[string]string) (*fieldValues, error) {
	fv := &fieldValues{skip: map[string]string{}, fields: map[string][]any{}, base: nil, localOnly: ""}
	maps.Copy(fv.skip, cliManagedFields)

	path := filepath.Join(fieldsDir, resourceType+".yml")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fv, nil
	}
	if err != nil {
		return nil, err
	}

	var missing string
	expanded := os.Expand(string(data), func(key string) string {
		value, ok := vars[key]
		if !ok {
			missing = key
		}
		return value
	})
	if missing != "" {
		return nil, fmt.Errorf("%s uses $%s, which this suite does not provide here", path, missing)
	}

	var file struct {
		Skip      map[string]string `yaml:"skip"`
		Fields    map[string][]any  `yaml:"fields"`
		Base      any               `yaml:"base"`
		LocalOnly string            `yaml:"local_only"`
	}
	if err := yaml.Unmarshal([]byte(expanded), &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	maps.Copy(fv.skip, file.Skip)
	maps.Copy(fv.fields, file.Fields)
	fv.base = file.Base
	fv.localOnly = file.LocalOnly

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

// defaultValues returns the values to try for a leaf field. Two per field is enough to
// observe a value-to-value transition on top of add and remove; more would multiply the
// matrix without testing a different code path.
func defaultValues(typ reflect.Type) []any {
	if values := enumValues(typ); values != nil {
		return values
	}
	return kindValues(typ.Kind())
}

// enumValues returns two values of an SDK enum, which declares its own: every generated
// enum type has a Values() method on its pointer receiver. Without this an enum field gets
// the generic "x" and "y", which a real backend rejects or silently ignores -- so the field
// reports nothing about whether the engine handles a change to it.
//
// The values are sorted because the SDK documents no order, and the report is a golden.
func enumValues(typ reflect.Type) []any {
	if typ.Kind() != reflect.String || typ == reflect.TypeFor[string]() {
		return nil
	}
	method, ok := reflect.PointerTo(typ).MethodByName("Values")
	if !ok || method.Type.NumIn() != 1 || method.Type.NumOut() != 1 {
		return nil
	}
	result := method.Func.Call([]reflect.Value{reflect.New(typ)})[0]
	if result.Kind() != reflect.Slice || result.Type().Elem() != typ {
		return nil
	}

	all := make([]string, 0, result.Len())
	for i := range result.Len() {
		if value := result.Index(i).String(); value != "" {
			all = append(all, value)
		}
	}
	slices.Sort(all)
	if len(all) < 2 {
		// One value cannot show a value-to-value move, and add and remove alone would say
		// nothing an enum-specific case does not already cover.
		return nil
	}
	return []any{reflect.ValueOf(all[0]).Convert(typ).Interface(), reflect.ValueOf(all[1]).Convert(typ).Interface()}
}

// kindValues are the fallback values for a plain Go kind.
//
//nolint:exhaustive // the default branch covers every kind without a generic value
func kindValues(kind reflect.Kind) []any {
	switch kind {
	case reflect.Bool:
		return []any{false, true}
	case reflect.String:
		return []any{"x", "y"}
	case reflect.Int, reflect.Int32, reflect.Int64:
		return []any{1, 2}
	case reflect.Float32, reflect.Float64:
		return []any{1.0, 2.0}
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

	path   string // structpath path, e.g. "comment" or "email_notifications.on_failure"
	kind   reflect.Kind
	values []any

	// required fields do not get an "absent" transition: a config missing one is
	// rejected by bundle validate, so removing it is not something a user can deploy.
	required bool
}

// transition is one move of a field from one value to another.
type transition struct {
	from, to any
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
		values = append([]any{absent}, values...)
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

// skipReason returns why a field is excluded, if it is. A key may be a pattern, matched the
// same way the planner matches its own field rules -- so "aliases[*].id" covers
// "aliases[0].id", and a trailing ".*" or "[*]" covers everything beneath a block that only
// works as a whole.
func (fv *fieldValues) skipReason(path string) (string, bool) {
	if reason, ok := fv.skip[path]; ok {
		return reason, true
	}

	concrete, err := structpath.ParsePath(path)
	if err != nil {
		// A pattern, which cannot be matched against another pattern: compare textually.
		for key, reason := range fv.skip {
			if strings.HasPrefix(path, key+".") || strings.HasPrefix(path, key+"[") {
				return reason, true
			}
		}
		return "", false
	}

	for key, reason := range fv.skip {
		pattern, err := structpath.ParsePattern(key)
		if err != nil {
			continue
		}
		if concrete.HasPatternPrefix(pattern) {
			return reason, true
		}
	}
	return "", false
}

func isRequired(resourceType, path string) bool {
	// The generated keys are patterns, so an index has to become a wildcard first:
	// "tasks[0].task_key" is declared under "resources.jobs.*.tasks[*]".
	parent, name := "", patternOf(path)
	if i := strings.LastIndex(name, "."); i >= 0 {
		parent, name = name[:i], name[i+1:]
	}
	return slices.Contains(requiredFields(resourceType, parent), name)
}

// enumerateFields walks the resource's input config type the way cmd/bundle/debug
// refschema does, and pairs each field with the values to try.
//
// resource is the deployed resource, which is what makes slices and maps testable. A slice or map the config populates becomes a field in its own right,
// with add-an-entry and remove-an-entry transitions; and a pattern like
// "tasks[*].description" is expanded to the indices that exist, so the fields inside an
// element get the same treatment as any other. A pattern with nothing behind it in the
// config is reported as not covered rather than silently tested against nothing.
func enumerateFields(resourceType string, inputType reflect.Type, fv *fieldValues, resource any, runSeed uint64, unsettable []dresources.FieldRule) (fields []field, wildcard []string, inertFields map[string]string) {
	inertFields = map[string]string{}
	add := func(path string, kind reflect.Kind, values []any) {
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
				for _, concrete := range expandPattern(resource, path) {
					add(concrete, typ.Kind(), containerValues(resource, concrete))
				}
			}
			return true
		}

		if !isWildcard {
			values := fv.fields[path]
			if values == nil {
				values = defaultValues(typ)
			}
			add(path, typ.Kind(), values)
			return false
		}

		concrete := expandPattern(resource, path)
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
				values = defaultValues(typ)
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

// containerValues returns the values to try for a slice or map field that the deployed
// resource actually populates: the resource's own value, and that value with one entry
// dropped. Combined with the implicit "absent", one field then covers adding and removing
// the whole container as well as adding and removing a single entry -- all with data the
// backend has already accepted, so nothing has to be invented.
//
// Both are deep copies: a later edit rewrites the same map or slice in place, which would
// otherwise change a value recorded here.
func containerValues(resource any, path string) []any {
	current, err := structaccess.GetByString(resource, path)
	if err != nil || isAbsent(current) {
		return nil
	}

	// Both values are deep copies. A shallow one would share the pointers inside an
	// element -- a task's notebook_task, an init script's workspace block -- and a later
	// edit to a field under that element would reach into a value recorded here.
	full, err := clone(current)
	if err != nil {
		return nil
	}
	trimmed, err := clone(current)
	if err != nil {
		return nil
	}

	value := reflect.ValueOf(trimmed)
	switch value.Kind() {
	case reflect.Slice:
		if value.Len() == 0 {
			return nil
		}
		trimmed = value.Slice(0, value.Len()-1).Interface()

	case reflect.Map:
		keys := sortedMapKeys(value)
		if len(keys) == 0 {
			return nil
		}
		value.SetMapIndex(keys[len(keys)-1], reflect.Value{})
		trimmed = value.Interface()

	default:
		return nil
	}

	return []any{full, trimmed}
}

// clone deep-copies a value through the JSON representation the API uses, which the SDK
// types define themselves -- so ForceSendFields survives the copy.
func clone(value any) (any, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	copied := reflect.New(reflect.TypeOf(value))
	if err := json.Unmarshal(body, copied.Interface()); err != nil {
		return nil, err
	}
	return copied.Elem().Interface(), nil
}

// sortedMapKeys orders a map's keys so that which entry gets dropped, and which entries a
// wildcard expands to, is the same on every run. A Go map has no order of its own.
func sortedMapKeys(value reflect.Value) []reflect.Value {
	keys := value.MapKeys()
	slices.SortFunc(keys, func(a, b reflect.Value) int { return strings.Compare(a.String(), b.String()) })
	return keys
}

// expandPattern turns a pattern from the type walk into the concrete paths the deployed
// resource actually has: "tasks[*].description" against a resource with one task yields
// "tasks[0].description". A pattern with nothing behind it yields nothing, which is how a
// field under an absent container is reported as not covered.
func expandPattern(resource any, pattern string) []string {
	paths := []string{""}
	for _, seg := range splitPattern(pattern) {
		var next []string
		for _, prefix := range paths {
			switch seg {
			case "[*]":
				value, err := valueAt(resource, prefix)
				if err != nil || value.Kind() != reflect.Slice {
					continue
				}
				for i := range value.Len() {
					next = append(next, prefix+"["+strconv.Itoa(i)+"]")
				}
			case "*":
				value, err := valueAt(resource, prefix)
				if err != nil || value.Kind() != reflect.Map {
					continue
				}
				for _, key := range sortedMapKeys(value) {
					next = append(next, prefix+"["+quoteKey(key.String())+"]")
				}
			default:
				next = append(next, joinPath(prefix, seg))
			}
		}
		paths = next
	}
	return paths
}

// valueAt reads a path off the resource, with "" meaning the resource itself.
func valueAt(resource any, path string) (reflect.Value, error) {
	if path == "" {
		return reflect.Indirect(reflect.ValueOf(resource)), nil
	}
	value, err := structaccess.GetByString(resource, path)
	if err != nil {
		return reflect.Value{}, err
	}
	if isAbsent(value) {
		return reflect.Value{}, fmt.Errorf("%s is absent", path)
	}
	return reflect.Indirect(reflect.ValueOf(value)), nil
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

// quoteKey renders a map key the way structpath parses it back.
func quoteKey(key string) string {
	return "'" + strings.ReplaceAll(key, "'", "''") + "'"
}
