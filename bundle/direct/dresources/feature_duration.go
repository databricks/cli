package dresources

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn/dynvar"
	sdkduration "github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

// durationComponent matches one leading "<number><unit>" component of a duration string, including
// the day (d) and week (w) units time.ParseDuration lacks.
var durationComponent = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)(ns|us|µs|ms|s|m|h|d|w)`)

const (
	hoursPerDay  = 24
	hoursPerWeek = 24 * 7
	// maxDurationHours is the largest whole-hour count representable as a time.Duration (an int64
	// nanosecond count).
	maxDurationHours = int64(math.MaxInt64) / int64(time.Hour)
)

// duration parses a duration composed of week, day, hour, minute and second units ("1w", "7d",
// "3d15m", "604800s") into the SDK type, which holds only seconds.
func duration(s string) (*sdkduration.Duration, error) {
	if parsed, ok := protobufDuration(s); ok {
		return parsed, nil
	}
	d, err := parseExtendedDuration(s)
	if err != nil {
		return nil, fmt.Errorf("invalid duration %q: %w; use e.g. 604800s, 168h, 7d or 3d15m", s, err)
	}
	return sdkduration.New(d), nil
}

// protobufDuration reports whether s is already a protobuf duration, returning the parsed value.
func protobufDuration(s string) (*sdkduration.Duration, bool) {
	quoted, err := json.Marshal(s)
	if err != nil {
		return nil, false
	}
	var parsed sdkduration.Duration
	if err := parsed.UnmarshalJSON(quoted); err != nil {
		return nil, false
	}
	return &parsed, true
}

// parseExtendedDuration parses a Go duration string extended with day (d, = 24h) and week (w, =
// 168h) units. Components in those units are summed here; all other components (h and below) are
// handed to time.ParseDuration unchanged, which keeps its exact integer handling for the common
// forms.
//
// Duplicates parseTTLDuration in cmd/workspace/postgres/overrides.go, which serves the postgres
// branch --ttl flag.
func parseExtendedDuration(s string) (time.Duration, error) {
	// Fast path: no day/week unit means time.ParseDuration handles it all. No other supported unit
	// contains 'd' or 'w', so this check is exact.
	if !strings.ContainsAny(s, "dw") {
		return time.ParseDuration(s)
	}

	rest := s
	neg := false
	switch {
	case strings.HasPrefix(rest, "-"):
		neg, rest = true, rest[1:]
	case strings.HasPrefix(rest, "+"):
		rest = rest[1:]
	}

	var extra time.Duration      // accumulated day/week components
	var residual strings.Builder // remaining components, parsed by time.ParseDuration
	for rest != "" {
		m := durationComponent.FindStringSubmatch(rest)
		if m == nil {
			return 0, fmt.Errorf("unrecognized duration component %q", rest)
		}
		switch m[2] {
		case "d", "w":
			hoursPerUnit := int64(hoursPerDay)
			if m[2] == "w" {
				hoursPerUnit = hoursPerWeek
			}
			if strings.Contains(m[1], ".") {
				f, err := strconv.ParseFloat(m[1], 64)
				if err != nil {
					return 0, err
				}
				ns := f * float64(hoursPerUnit) * float64(time.Hour)
				if ns > float64(math.MaxInt64) {
					return 0, fmt.Errorf("duration component %s%s overflows", m[1], m[2])
				}
				extra += time.Duration(ns)
			} else {
				n, err := strconv.ParseInt(m[1], 10, 64)
				if err != nil {
					return 0, err
				}
				if n > maxDurationHours/hoursPerUnit {
					return 0, fmt.Errorf("duration component %s%s overflows", m[1], m[2])
				}
				extra += time.Duration(n*hoursPerUnit) * time.Hour
			}
		default:
			residual.WriteString(m[1])
			residual.WriteString(m[2])
		}
		rest = rest[len(m[0]):]
	}

	total := extra
	if residual.Len() > 0 {
		d, err := time.ParseDuration(residual.String())
		if err != nil {
			return 0, err
		}
		total += d
	}
	if neg {
		total = -total
	}
	return total, nil
}

// optionalDuration converts a configured duration into the API type, leaving an unset value and an
// unresolved reference alone.
func optionalDuration(s string) (*sdkduration.Duration, error) {
	if s == "" || dynvar.IsPureVariableReference(s) {
		return nil, nil
	}
	return duration(s)
}

// optionalCanonicalDuration renders a duration the API types as a string in the seconds form the
// API answers with, under the same two exemptions as optionalDuration.
func optionalCanonicalDuration(s string) (string, error) {
	if s == "" || dynvar.IsPureVariableReference(s) {
		return s, nil
	}
	parsed, err := duration(s)
	if err != nil {
		return "", err
	}
	marshalled, err := parsed.MarshalJSON()
	if err != nil {
		return "", err
	}
	var canonical string
	if err := json.Unmarshal(marshalled, &canonical); err != nil {
		return "", err
	}
	return canonical, nil
}

// durationConversion is one duration field's conversion: the API field name, which names it in the
// error, and the assignment itself.
type durationConversion struct {
	field string
	apply func() error
}

// shadowedDuration parses the string the bundle config shadows an SDK duration field with.
func shadowedDuration(field, source string, dest **sdkduration.Duration) durationConversion {
	return durationConversion{field: field, apply: func() error {
		parsed, err := optionalDuration(source)
		if err != nil {
			return err
		}
		*dest = parsed
		return nil
	}}
}

// inPlaceDuration rewrites a duration the SDK types as a string, which the config does not shadow.
func inPlaceDuration(field string, dest *string) durationConversion {
	return durationConversion{field: field, apply: func() error {
		rewritten, err := optionalCanonicalDuration(*dest)
		if err != nil {
			return err
		}
		*dest = rewritten
		return nil
	}}
}

// convertDurations applies a registry, naming the field that failed.
//
// A field whose SDK type flips between *sdkduration.Duration and string has to move between
// shadowedDuration and inPlaceDuration, which take different pointer types, so the compiler reports
// it. A duration field the API adds is reported by TestFeatureDurationRegistryIsComplete.
func convertDurations(conversions []durationConversion) error {
	for _, conversion := range conversions {
		if err := conversion.apply(); err != nil {
			return fmt.Errorf("%s: %w", conversion.field, err)
		}
	}
	return nil
}

// windowState copies the API type a window's config wrapper embeds and converts its durations.
// embedded returns that copy because Go does not permit field access on a type parameter.
func windowState[Config, State any](
	input *Config,
	embedded func(*Config) State,
	durations func(*Config, *State) []durationConversion,
) (*State, error) {
	if input == nil {
		return nil, nil
	}
	state := embedded(input)
	if err := convertDurations(durations(input, &state)); err != nil {
		return nil, err
	}
	return &state, nil
}

// The registry: every duration field the adapter converts, one entry per field. A spec change that
// adds a duration field is one line here.

func featureSourceLatenessDurations(input *resources.FeatureSourceLateness, state *ml.SourceLateness) []durationConversion {
	return []durationConversion{
		shadowedDuration("settling_delay", input.SettlingDelay, &state.SettlingDelay),
	}
}

func rollingWindowDurations(input *resources.FeatureRollingWindow, state *ml.RollingWindow) []durationConversion {
	return []durationConversion{
		shadowedDuration("delay", input.Delay, &state.Delay),
		shadowedDuration("window_duration", input.WindowDuration, &state.WindowDuration),
	}
}

func sawtoothWindowDurations(input *resources.FeatureSawtoothWindow, state *ml.SawtoothWindow) []durationConversion {
	return []durationConversion{
		shadowedDuration("delay", input.Delay, &state.Delay),
		shadowedDuration("window_duration", input.WindowDuration, &state.WindowDuration),
	}
}

func slidingWindowDurations(input *resources.FeatureSlidingWindow, state *ml.SlidingWindow) []durationConversion {
	return []durationConversion{
		shadowedDuration("delay", input.Delay, &state.Delay),
		shadowedDuration("offset", input.Offset, &state.Offset),
		inPlaceDuration("window_duration", &state.WindowDuration),
		inPlaceDuration("slide_duration", &state.SlideDuration),
	}
}

func tumblingWindowDurations(input *resources.FeatureTumblingWindow, state *ml.TumblingWindow) []durationConversion {
	return []durationConversion{
		shadowedDuration("delay", input.Delay, &state.Delay),
		shadowedDuration("offset", input.Offset, &state.Offset),
		inPlaceDuration("window_duration", &state.WindowDuration),
	}
}

// The config does not shadow a continuous window's durations, so there is nothing to read from it.
func continuousWindowDurations(_, state *ml.ContinuousWindow) []durationConversion {
	return []durationConversion{
		inPlaceDuration("offset", &state.Offset),
		inPlaceDuration("window_duration", &state.WindowDuration),
	}
}

// featureState converts the bundle config for a feature into the API type, parsing every duration.
// The config declares durations as strings; see the override note in
// bundle/config/resources/feature.go.
func featureState(input *resources.Feature) (*ml.Feature, error) {
	state := input.Feature

	function, err := featureFunctionState(&input.Function)
	if err != nil {
		return nil, fmt.Errorf("function: %w", err)
	}
	state.Function = *function

	source, err := featureDataSourceState(&input.Source)
	if err != nil {
		return nil, fmt.Errorf("source: %w", err)
	}
	state.Source = *source

	// The deprecated top-level time_window is converted on the same terms as the one under the
	// aggregation function.
	state.TimeWindow, err = featureTimeWindowState(input.TimeWindow)
	if err != nil {
		return nil, fmt.Errorf("time_window: %w", err)
	}

	return &state, nil
}

func featureFunctionState(input *resources.FeatureFunction) (*ml.Function, error) {
	state := input.Function

	aggregation, err := featureAggregationFunctionState(input.AggregationFunction)
	if err != nil {
		return nil, fmt.Errorf("aggregation_function: %w", err)
	}
	state.AggregationFunction = aggregation

	return &state, nil
}

func featureAggregationFunctionState(input *resources.FeatureAggregationFunction) (*ml.AggregationFunction, error) {
	if input == nil {
		return nil, nil
	}
	state := input.AggregationFunction

	timeWindow, err := featureTimeWindowState(input.TimeWindow)
	if err != nil {
		return nil, fmt.Errorf("time_window: %w", err)
	}
	state.TimeWindow = timeWindow

	return &state, nil
}

func featureTimeWindowState(input *resources.FeatureTimeWindow) (*ml.TimeWindow, error) {
	if input == nil {
		return nil, nil
	}
	state := input.TimeWindow

	var err error
	if state.Rolling, err = windowState(input.Rolling,
		func(c *resources.FeatureRollingWindow) ml.RollingWindow { return c.RollingWindow },
		rollingWindowDurations); err != nil {
		return nil, fmt.Errorf("rolling: %w", err)
	}
	if state.Sawtooth, err = windowState(input.Sawtooth,
		func(c *resources.FeatureSawtoothWindow) ml.SawtoothWindow { return c.SawtoothWindow },
		sawtoothWindowDurations); err != nil {
		return nil, fmt.Errorf("sawtooth: %w", err)
	}
	if state.Sliding, err = windowState(input.Sliding,
		func(c *resources.FeatureSlidingWindow) ml.SlidingWindow { return c.SlidingWindow },
		slidingWindowDurations); err != nil {
		return nil, fmt.Errorf("sliding: %w", err)
	}
	if state.Tumbling, err = windowState(input.Tumbling,
		func(c *resources.FeatureTumblingWindow) ml.TumblingWindow { return c.TumblingWindow },
		tumblingWindowDurations); err != nil {
		return nil, fmt.Errorf("tumbling: %w", err)
	}
	if state.Continuous, err = windowState(state.Continuous,
		func(c *ml.ContinuousWindow) ml.ContinuousWindow { return *c },
		continuousWindowDurations); err != nil {
		return nil, fmt.Errorf("continuous: %w", err)
	}

	return &state, nil
}

func featureDataSourceState(input *resources.FeatureDataSource) (*ml.DataSource, error) {
	state := input.DataSource

	lateness, err := featureSourceLatenessState(input.Lateness)
	if err != nil {
		return nil, fmt.Errorf("lateness: %w", err)
	}
	state.Lateness = lateness

	return &state, nil
}

func featureSourceLatenessState(input *resources.FeatureSourceLateness) (*ml.SourceLateness, error) {
	if input == nil {
		return nil, nil
	}
	state := input.SourceLateness

	if err := convertDurations(featureSourceLatenessDurations(input, &state)); err != nil {
		return nil, err
	}

	return &state, nil
}
