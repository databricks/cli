package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	sdkduration "github.com/databricks/databricks-sdk-go/common/types/duration"
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

// featureWithEveryDuration sets every duration field a feature has to one day, so
// TestFeatureStateConvertsEveryWindow can assert each one survives conversion.
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
