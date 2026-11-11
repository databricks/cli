package aircmd

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// metric builds one epoch-metric history point: fractional epoch as the value,
// global step as the MLflow step, wall-clock as the timestamp (ms).
func metric(tsMillis, step int64, epoch float64) ml.Metric {
	return ml.Metric{Key: "epoch", Timestamp: tsMillis, Step: step, Value: epoch}
}

func TestComputeETA_StepMode(t *testing.T) {
	// max_steps wins over num_train_epochs. 4000 steps in 100s => 40 steps/s;
	// 5000 steps remain => 125s; 5000/10000 => 50%.
	eta := computeETA(
		map[string]string{"max_steps": "10000", "num_train_epochs": "3"},
		[]ml.Metric{metric(0, 1000, 0.3), metric(100_000, 5000, 1.5)},
	)
	require.NotNil(t, eta)
	assert.Equal(t, int64(125), eta.RemainingSeconds)
	assert.Equal(t, 50, eta.PercentComplete)
	assert.Equal(t, "50% · ~2m left", eta.detailed())
	assert.Equal(t, "50% · ~2m", eta.compact())
}

func TestComputeETA_EpochMode(t *testing.T) {
	// No usable max_steps (Trainer logs -1 when training by epochs), so the
	// fractional epoch drives it: 1.0 epoch in 90s => 1.5 remain of 3 => 135s;
	// 1.5/3 => 50%.
	eta := computeETA(
		map[string]string{"max_steps": "-1", "num_train_epochs": "3"},
		[]ml.Metric{metric(0, 100, 0.5), metric(90_000, 900, 1.5)},
	)
	require.NotNil(t, eta)
	assert.Equal(t, int64(135), eta.RemainingSeconds)
	assert.Equal(t, 50, eta.PercentComplete)
	assert.Equal(t, "50% · ~2m left", eta.detailed())
}

func TestComputeETA_UsesTrailingWindow(t *testing.T) {
	// Early points are slow, recent points fast; the window should reflect the
	// recent rate, not the whole-run average. Build 12 points; only the last
	// etaWindowPoints (10) count.
	var pts []ml.Metric
	pts = append(pts, metric(0, 0, 0), metric(100_000, 10, 0.1))
	for i := int64(1); i <= 10; i++ {
		pts = append(pts, metric(100_000+i*10_000, 10+i*100, 0.1+float64(i)*0.1))
	}
	eta := computeETA(map[string]string{"max_steps": "2000"}, pts)
	require.NotNil(t, eta)
	// Windowed rate is 10 steps/s; current step 1010, so 990 remain => 99s.
	assert.Equal(t, int64(99), eta.RemainingSeconds)
	assert.Equal(t, 51, eta.PercentComplete) // 1010/2000 = 50.5 -> 51
}

func TestComputeETA_NoEstimate(t *testing.T) {
	base := []ml.Metric{metric(0, 100, 0.3), metric(100_000, 5000, 1.5)}
	cases := []struct {
		name    string
		params  map[string]string
		history []ml.Metric
	}{
		{"no total param", map[string]string{}, base},
		{"max_steps not positive", map[string]string{"max_steps": "-1"}, base},
		{"num_train_epochs zero", map[string]string{"num_train_epochs": "0"}, base},
		{"total param unparseable", map[string]string{"max_steps": "lots"}, base},
		{"too few points", map[string]string{"max_steps": "10000"}, base[:1]},
		{"still warming up", map[string]string{"max_steps": "10000"}, []ml.Metric{metric(0, 2, 0.01), metric(30_000, 8, 0.02)}},
		{"no elapsed time", map[string]string{"max_steps": "10000"}, []ml.Metric{metric(500, 100, 0.3), metric(500, 5000, 1.5)}},
		{"already at max steps", map[string]string{"max_steps": "5000"}, base},
		{"epoch already complete", map[string]string{"num_train_epochs": "1.5"}, base},
		{"step not advancing", map[string]string{"max_steps": "10000"}, []ml.Metric{metric(0, 5000, 1.5), metric(100_000, 5000, 1.5)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Nil(t, computeETA(tc.params, tc.history))
		})
	}
}

func TestComputeETA_PercentClampedBelow100(t *testing.T) {
	// A run one step from done is still "running", so it never shows 100%.
	eta := computeETA(
		map[string]string{"max_steps": "1000"},
		[]ml.Metric{metric(0, 900, 0.9), metric(10_000, 999, 0.999)},
	)
	require.NotNil(t, eta)
	assert.Equal(t, 99, eta.PercentComplete)
}

func TestTrainingETADisplay(t *testing.T) {
	eta := &trainingETA{RemainingSeconds: 2900, PercentComplete: 81}
	assert.Equal(t, "81% · ~48m left", eta.detailed())
	assert.Equal(t, "81% · ~48m", eta.compact())
}

func TestCoarseDuration(t *testing.T) {
	cases := []struct {
		seconds int64
		want    string
	}{
		{0, "<1m"},
		{20, "<1m"},
		{80, "1m"},
		{125, "2m"},
		{3600, "1h"},
		{8100, "2h 15m"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, coarseDuration(tc.seconds))
	}
}
