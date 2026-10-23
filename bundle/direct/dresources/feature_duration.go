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
	if state.Rolling, err = rollingWindowState(input.Rolling); err != nil {
		return nil, fmt.Errorf("rolling: %w", err)
	}
	if state.Sawtooth, err = sawtoothWindowState(input.Sawtooth); err != nil {
		return nil, fmt.Errorf("sawtooth: %w", err)
	}
	if state.Sliding, err = slidingWindowState(input.Sliding); err != nil {
		return nil, fmt.Errorf("sliding: %w", err)
	}
	if state.Tumbling, err = tumblingWindowState(input.Tumbling); err != nil {
		return nil, fmt.Errorf("tumbling: %w", err)
	}
	if state.Continuous, err = continuousWindowState(state.Continuous); err != nil {
		return nil, fmt.Errorf("continuous: %w", err)
	}

	return &state, nil
}

func rollingWindowState(input *resources.FeatureRollingWindow) (*ml.RollingWindow, error) {
	if input == nil {
		return nil, nil
	}
	state := input.RollingWindow

	var err error
	if state.Delay, err = optionalDuration(input.Delay); err != nil {
		return nil, fmt.Errorf("delay: %w", err)
	}
	if state.WindowDuration, err = optionalDuration(input.WindowDuration); err != nil {
		return nil, fmt.Errorf("window_duration: %w", err)
	}

	return &state, nil
}

func sawtoothWindowState(input *resources.FeatureSawtoothWindow) (*ml.SawtoothWindow, error) {
	if input == nil {
		return nil, nil
	}
	state := input.SawtoothWindow

	var err error
	if state.Delay, err = optionalDuration(input.Delay); err != nil {
		return nil, fmt.Errorf("delay: %w", err)
	}
	if state.WindowDuration, err = optionalDuration(input.WindowDuration); err != nil {
		return nil, fmt.Errorf("window_duration: %w", err)
	}

	return &state, nil
}

func slidingWindowState(input *resources.FeatureSlidingWindow) (*ml.SlidingWindow, error) {
	if input == nil {
		return nil, nil
	}
	state := input.SlidingWindow

	var err error
	if state.Delay, err = optionalDuration(input.Delay); err != nil {
		return nil, fmt.Errorf("delay: %w", err)
	}
	if state.Offset, err = optionalDuration(input.Offset); err != nil {
		return nil, fmt.Errorf("offset: %w", err)
	}
	if state.WindowDuration, err = optionalCanonicalDuration(state.WindowDuration); err != nil {
		return nil, fmt.Errorf("window_duration: %w", err)
	}
	if state.SlideDuration, err = optionalCanonicalDuration(state.SlideDuration); err != nil {
		return nil, fmt.Errorf("slide_duration: %w", err)
	}

	return &state, nil
}

func tumblingWindowState(input *resources.FeatureTumblingWindow) (*ml.TumblingWindow, error) {
	if input == nil {
		return nil, nil
	}
	state := input.TumblingWindow

	var err error
	if state.Delay, err = optionalDuration(input.Delay); err != nil {
		return nil, fmt.Errorf("delay: %w", err)
	}
	if state.Offset, err = optionalDuration(input.Offset); err != nil {
		return nil, fmt.Errorf("offset: %w", err)
	}
	if state.WindowDuration, err = optionalCanonicalDuration(state.WindowDuration); err != nil {
		return nil, fmt.Errorf("window_duration: %w", err)
	}

	return &state, nil
}

func continuousWindowState(input *ml.ContinuousWindow) (*ml.ContinuousWindow, error) {
	if input == nil {
		return nil, nil
	}
	state := *input

	var err error
	if state.Offset, err = optionalCanonicalDuration(state.Offset); err != nil {
		return nil, fmt.Errorf("offset: %w", err)
	}
	if state.WindowDuration, err = optionalCanonicalDuration(state.WindowDuration); err != nil {
		return nil, fmt.Errorf("window_duration: %w", err)
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

	var err error
	if state.SettlingDelay, err = optionalDuration(input.SettlingDelay); err != nil {
		return nil, fmt.Errorf("settling_delay: %w", err)
	}

	return &state, nil
}
