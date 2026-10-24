package dresources

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structtag"
	sdkduration "github.com/databricks/databricks-sdk-go/common/types/duration"
	sdkfieldmask "github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExtendedDuration(t *testing.T) {
	for _, tc := range []struct {
		input   string
		want    string // canonical protobuf form
		wantErr bool
	}{
		{input: "604800s", want: "604800s"},
		{input: "168h", want: "604800s"},
		{input: "7d", want: "604800s"},
		{input: "1w", want: "604800s"},
		{input: "3d15m", want: "260100s"},
		{input: "1.5d", want: "129600s"},
		{input: "0.5s", want: "0.500s"},
		{input: "", want: ""},
		{input: "abc", wantErr: true},
		{input: "7x", wantErr: true},
		{input: "7dd", wantErr: true},
		{input: "100000000000w", wantErr: true},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := optionalCanonicalDuration(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestOptionalCanonicalDurationRendersProtobufForm pins that a value the API already accepts is
// still rendered the way the API renders it, so state matches what the server answers with.
func TestOptionalCanonicalDurationRendersProtobufForm(t *testing.T) {
	for input, want := range map[string]string{
		"604800s":      "604800s",
		"0.5s":         "0.500s",
		"0.500s":       "0.500s",
		"1.000000001s": "1.000000001s",
		"0s":           "0s",
	} {
		got, err := optionalCanonicalDuration(input)
		require.NoError(t, err, input)
		assert.Equal(t, want, got, input)
	}
}

func seconds(t *testing.T, d *sdkduration.Duration) float64 {
	t.Helper()
	require.NotNil(t, d)
	return d.AsDuration().Seconds()
}

func TestFeatureStateConvertsEveryWindow(t *testing.T) {
	state, err := featureState(featureWithEveryDuration())
	require.NoError(t, err)

	window := state.Function.AggregationFunction.TimeWindow
	assert.Equal(t, 86400.0, seconds(t, window.Rolling.WindowDuration))
	assert.Equal(t, 86400.0, seconds(t, window.Rolling.Delay))
	assert.Equal(t, 86400.0, seconds(t, window.Sawtooth.WindowDuration))
	assert.Equal(t, 86400.0, seconds(t, window.Sliding.Offset))
	assert.Equal(t, "86400s", window.Sliding.WindowDuration)
	assert.Equal(t, "86400s", window.Sliding.SlideDuration)
	assert.Equal(t, "86400s", window.Tumbling.WindowDuration)
	assert.Equal(t, "86400s", window.Continuous.WindowDuration)
	assert.Equal(t, 86400.0, seconds(t, state.Source.Lateness.SettlingDelay))

	// The deprecated top-level time_window is converted on the same terms.
	assert.Equal(t, 86400.0, seconds(t, state.TimeWindow.Rolling.WindowDuration))
}

func TestFeatureStateAcceptsCompoundAndProtobufForms(t *testing.T) {
	input := &resources.Feature{}
	input.Function.AggregationFunction = &resources.FeatureAggregationFunction{
		TimeWindow: &resources.FeatureTimeWindow{
			Sawtooth: &resources.FeatureSawtoothWindow{WindowDuration: "3d15m"},
			Sliding:  &resources.FeatureSlidingWindow{},
		},
	}
	input.Function.AggregationFunction.TimeWindow.Sliding.SlideDuration = "604800s"

	state, err := featureState(input)
	require.NoError(t, err)
	window := state.Function.AggregationFunction.TimeWindow
	assert.Equal(t, 260100.0, seconds(t, window.Sawtooth.WindowDuration))
	assert.Equal(t, "604800s", window.Sliding.SlideDuration)
}

func TestFeatureStateLeavesUnsetAndVariableReferences(t *testing.T) {
	input := &resources.Feature{}
	input.Function.AggregationFunction = &resources.FeatureAggregationFunction{
		TimeWindow: &resources.FeatureTimeWindow{
			Rolling: &resources.FeatureRollingWindow{WindowDuration: "${var.window}"},
			Sliding: &resources.FeatureSlidingWindow{},
		},
	}

	state, err := featureState(input)
	require.NoError(t, err)
	window := state.Function.AggregationFunction.TimeWindow
	assert.Nil(t, window.Rolling.WindowDuration, "an unresolved reference is left alone")
	assert.Nil(t, window.Rolling.Delay, "an unset duration stays unset")
	assert.Empty(t, window.Sliding.WindowDuration)
}

func TestFeatureStateWithoutWindows(t *testing.T) {
	state, err := featureState(&resources.Feature{})
	require.NoError(t, err)
	assert.Nil(t, state.Function.AggregationFunction)
	assert.Nil(t, state.Source.Lateness)
	assert.Nil(t, state.TimeWindow)
}

func TestFeaturePrepareInputConfigReportsBadDuration(t *testing.T) {
	input := &resources.Feature{}
	input.Function.AggregationFunction = &resources.FeatureAggregationFunction{
		TimeWindow: &resources.FeatureTimeWindow{
			Rolling: &resources.FeatureRollingWindow{WindowDuration: "7 fortnights"},
		},
	}

	_, err := (*ResourceFeature)(nil).PrepareInputConfig(input, "resources.features.f")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "function: aggregation_function: time_window: rolling: window_duration:")
	assert.Contains(t, err.Error(), `"7 fortnights"`)
}

func TestFeaturePrepareInputConfigPassesConfigThrough(t *testing.T) {
	input := featureWithEveryDuration()
	got, err := (*ResourceFeature)(nil).PrepareInputConfig(input, "resources.features.f")
	require.NoError(t, err)
	assert.Same(t, input, got.Value)
	assert.Nil(t, got.Refs)
}

// ---- the guard ----

// durationNamed reports whether a string-typed field holds a duration. The API types five of these
// as plain strings, and neither the Go type nor the generated schema says so, so the name is the
// only signal.
func durationNamed(jsonName string) bool {
	leaf := jsonName
	if i := strings.LastIndex(leaf, "."); i >= 0 {
		leaf = leaf[i+1:]
	}
	// freshness_target is the one duration-shaped name that follows no convention, so it is named.
	return strings.HasSuffix(leaf, "_duration") || strings.HasSuffix(leaf, "delay") ||
		leaf == "offset" || leaf == "freshness_target"
}

var durationType = reflect.TypeFor[sdkduration.Duration]()

func terminalStruct(typ reflect.Type) bool {
	return typ == durationType || typ == reflect.TypeFor[sdktime.Time]() ||
		typ == reflect.TypeFor[sdkfieldmask.FieldMask]()
}

// flattenedFields returns typ's fields the way encoding/json sees them: breadth-first from the
// outer struct inward, with a JSON name declared at a shallower depth hiding a deeper one, matching
// buildStructInfo in libs/dyn/convert/struct_info.go.
func flattenedFields(typ reflect.Type) []reflect.StructField {
	var out []reflect.StructField
	seen := map[string]bool{}
	queue := []reflect.Type{typ}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		var embedded []reflect.Type
		for i := range cur.NumField() {
			field := cur.Field(i)
			name := structtag.JSONTag(field.Tag.Get("json")).Name()

			if field.Anonymous && name == "" {
				inner := field.Type
				for inner.Kind() == reflect.Pointer {
					inner = inner.Elem()
				}
				if inner.Kind() == reflect.Struct {
					embedded = append(embedded, inner)
					continue
				}
			}
			if !field.IsExported() || name == "" || name == "-" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, field)
		}
		queue = append(queue, embedded...)
	}
	return out
}

// jsonLeaves maps the JSON path of every leaf reachable from typ to the leaf's Go type.
func jsonLeaves(typ reflect.Type) map[string]reflect.Type {
	out := map[string]reflect.Type{}
	collectJSONLeaves(typ, "", map[reflect.Type]bool{}, out)
	return out
}

func collectJSONLeaves(typ reflect.Type, prefix string, onPath map[reflect.Type]bool, out map[string]reflect.Type) {
	for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct || onPath[typ] {
		return
	}
	onPath[typ] = true
	defer delete(onPath, typ)

	for _, field := range flattenedFields(typ) {
		path := structtag.JSONTag(field.Tag.Get("json")).Name()
		if prefix != "" {
			path = prefix + "." + path
		}

		inner := field.Type
		for inner.Kind() == reflect.Pointer || inner.Kind() == reflect.Slice {
			inner = inner.Elem()
		}
		if inner.Kind() == reflect.Struct && !terminalStruct(inner) {
			collectJSONLeaves(inner, path, onPath, out)
			continue
		}
		out[path] = field.Type
	}
}

func isDurationTyped(typ reflect.Type) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ == durationType
}

// featureDurationRegistries returns every registry keyed by the json path its fields hang from in
// ml.Feature. Each window is reached by two paths, the deprecated top-level time_window and the one
// under the aggregation function, so each window registry is listed under both.
func featureDurationRegistries() map[string][]durationConversion {
	windows := map[string][]durationConversion{
		"rolling":    rollingWindowDurations(&resources.FeatureRollingWindow{}, &ml.RollingWindow{}),
		"sawtooth":   sawtoothWindowDurations(&resources.FeatureSawtoothWindow{}, &ml.SawtoothWindow{}),
		"sliding":    slidingWindowDurations(&resources.FeatureSlidingWindow{}, &ml.SlidingWindow{}),
		"tumbling":   tumblingWindowDurations(&resources.FeatureTumblingWindow{}, &ml.TumblingWindow{}),
		"continuous": continuousWindowDurations(nil, &ml.ContinuousWindow{}),
	}

	registries := map[string][]durationConversion{
		"source.lateness": featureSourceLatenessDurations(&resources.FeatureSourceLateness{}, &ml.SourceLateness{}),
	}
	for _, timeWindow := range []string{"time_window", "function.aggregation_function.time_window"} {
		for window, conversions := range windows {
			registries[timeWindow+"."+window] = conversions
		}
	}
	return registries
}

// TestFeatureDurationRegistryIsComplete fails when the API gains a duration field, or a window
// carrying one, that no registry converts, and when a registry still names a field the API dropped.
// A field whose SDK type flips between *sdkduration.Duration and string is reported by the compiler
// instead, because shadowedDuration and inPlaceDuration take different pointer types.
func TestFeatureDurationRegistryIsComplete(t *testing.T) {
	registered := map[string]bool{}
	for path, conversions := range featureDurationRegistries() {
		for _, conversion := range conversions {
			registered[path+"."+conversion.field] = true
		}
	}

	found := 0
	for path, typ := range jsonLeaves(reflect.TypeFor[ml.Feature]()) {
		if !isDurationTyped(typ) && !(typ.Kind() == reflect.String && durationNamed(path)) {
			continue
		}
		found++
		assert.Truef(t, registered[path],
			"ml.Feature has duration field %q that no registry converts; add one line to the registry "+
				"in bundle/direct/dresources/feature_duration.go", path)
		delete(registered, path)
	}
	assert.Positive(t, found, "walked no duration fields, the walker is broken")

	var stale []string
	for path := range registered {
		stale = append(stale, path)
	}
	assert.Emptyf(t, stale, "the registry converts fields ml.Feature does not have: %v", stale)
}

// TestFeatureDurationFieldsAreRetyped fails when the API gains a duration field the bundle config
// does not re-type as a string, including one on a window kind that does not exist yet.
func TestFeatureDurationFieldsAreRetyped(t *testing.T) {
	sdk := jsonLeaves(reflect.TypeFor[ml.Feature]())
	config := jsonLeaves(reflect.TypeFor[resources.Feature]())

	found := 0
	for path, typ := range sdk {
		if !isDurationTyped(typ) {
			continue
		}
		found++
		configType, ok := config[path]
		require.Truef(t, ok, "ml.Feature has duration field %q, missing from resources.Feature", path)
		assert.Equalf(t, reflect.String, configType.Kind(),
			"%q must be re-typed as a string in resources.Feature so a bundle can write 7d; "+
				"add a wrapper in bundle/config/resources/feature.go and convert it in featureState",
			path)
	}
	assert.Positive(t, found, "walked no duration fields, the walker is broken")
}

// TestFeatureShadowsAreNotStale fails when a re-typed field no longer matches a field of the API
// type, which is what an upstream rename leaves behind.
func TestFeatureShadowsAreNotStale(t *testing.T) {
	sdkFields := map[string]bool{}
	for _, field := range flattenedFields(reflect.TypeFor[ml.Feature]()) {
		sdkFields[structtag.JSONTag(field.Tag.Get("json")).Name()] = true
	}

	// The subtrees resources.Feature re-types are those whose top-level name it redeclares.
	shadowed := map[string]bool{}
	configType := reflect.TypeFor[resources.Feature]()
	for i := range configType.NumField() {
		field := configType.Field(i)
		name := structtag.JSONTag(field.Tag.Get("json")).Name()
		if !field.Anonymous && sdkFields[name] {
			shadowed[name] = true
		}
	}
	require.NotEmpty(t, shadowed, "found no re-typed subtree, the walker is broken")

	sdk := jsonLeaves(reflect.TypeFor[ml.Feature]())
	for path := range jsonLeaves(configType) {
		if !shadowed[strings.SplitN(path, ".", 2)[0]] {
			continue
		}
		_, ok := sdk[path]
		assert.Truef(t, ok, "resources.Feature declares %q but ml.Feature does not; a re-typed field is stale", path)
	}
}

// TestFeatureStateConvertsEveryDurationField is the coverage guard: every duration field the API
// declares must come out of featureState holding the configured value. A field that is re-typed but
// never converted arrives nil and fails here.
func TestFeatureStateConvertsEveryDurationField(t *testing.T) {
	state, err := featureState(featureWithEveryDuration())
	require.NoError(t, err)

	checked := 0
	for path, typ := range jsonLeaves(reflect.TypeFor[ml.Feature]()) {
		if !isDurationTyped(typ) && !(typ.Kind() == reflect.String && durationNamed(path)) {
			continue
		}
		checked++
		value := leafByPath(t, reflect.ValueOf(state), path)
		require.Truef(t, value.IsValid(), "%q is unreachable in the converted state", path)

		if isDurationTyped(typ) {
			require.Falsef(t, value.IsNil(),
				"%q came out unset: add it to featureState and to featureWithEveryDuration", path)
			assert.Equalf(t, 86400.0, value.Interface().(*sdkduration.Duration).AsDuration().Seconds(), path)
			continue
		}
		assert.Equalf(t, "86400s", value.String(),
			"%q came out unconverted: add it to featureState and to featureWithEveryDuration", path)
	}
	assert.Positive(t, checked, "walked no duration fields, the walker is broken")
}

// TestMaterializationSetDurationFieldsAreKnown fails when the API gives a materialization set a
// second duration-shaped field. freshness_target has no reader in the service, so the resource
// deliberately converts nothing; a new field forces that decision to be revisited.
func TestMaterializationSetDurationFieldsAreKnown(t *testing.T) {
	var found []string
	for path, typ := range jsonLeaves(reflect.TypeFor[ml.MaterializationSet]()) {
		if isDurationTyped(typ) || (typ.Kind() == reflect.String && durationNamed(path)) {
			found = append(found, path)
		}
	}
	assert.ElementsMatch(t, []string{"materialized_features.streaming_mode.freshness_target"}, found)
}

// leafByPath resolves a dotted JSON path against a struct value, following the same shadowing rule
// as flattenedFields.
func leafByPath(t *testing.T, v reflect.Value, path string) reflect.Value {
	t.Helper()
	for _, segment := range strings.Split(path, ".") {
		for v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return reflect.Value{}
		}
		var next reflect.Value
		for _, field := range flattenedFields(v.Type()) {
			if structtag.JSONTag(field.Tag.Get("json")).Name() != segment {
				continue
			}
			next = v.FieldByIndex(field.Index)
			break
		}
		if !next.IsValid() {
			return reflect.Value{}
		}
		v = next
	}
	return v
}

// featureWithEveryDuration sets every duration field a feature has to one day, so the coverage
// guard can assert each one survives conversion.
func featureWithEveryDuration() *resources.Feature {
	const day = "1d"

	window := func() *resources.FeatureTimeWindow {
		w := &resources.FeatureTimeWindow{
			Rolling:  &resources.FeatureRollingWindow{Delay: day, WindowDuration: day},
			Sawtooth: &resources.FeatureSawtoothWindow{Delay: day, WindowDuration: day},
			Sliding:  &resources.FeatureSlidingWindow{Delay: day, Offset: day},
			Tumbling: &resources.FeatureTumblingWindow{Delay: day, Offset: day},
		}
		w.Sliding.WindowDuration = day
		w.Sliding.SlideDuration = day
		w.Tumbling.WindowDuration = day
		w.Continuous = &ml.ContinuousWindow{Offset: day, WindowDuration: day}
		return w
	}

	input := &resources.Feature{
		Function: resources.FeatureFunction{
			AggregationFunction: &resources.FeatureAggregationFunction{TimeWindow: window()},
		},
		Source:     resources.FeatureDataSource{Lateness: &resources.FeatureSourceLateness{SettlingDelay: day}},
		TimeWindow: window(),
	}
	return input
}
