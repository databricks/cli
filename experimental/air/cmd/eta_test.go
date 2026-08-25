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
	// 5000 steps remain => 125s.
	eta := computeETA(
		map[string]string{"max_steps": "10000", "num_train_epochs": "3"},
		[]ml.Metric{metric(0, 1000, 0.3), metric(100_000, 5000, 1.5)},
	)
	require.NotNil(t, eta)
	assert.Equal(t, int64(125), eta.RemainingSeconds)
	assert.Equal(t, "step 5000/10000", eta.Progress)
}

func TestComputeETA_EpochMode(t *testing.T) {
	// No usable max_steps (Trainer logs -1 when training by epochs), so the
	// fractional epoch drives it: 1.0 epoch in 100s => 0.01 epoch/s; 1.5 remain
	// of 3 => 150s.
	eta := computeETA(
		map[string]string{"max_steps": "-1", "num_train_epochs": "3"},
		[]ml.Metric{metric(0, 100, 0.5), metric(100_000, 900, 1.5)},
	)
	require.NotNil(t, eta)
	assert.Equal(t, int64(150), eta.RemainingSeconds)
	assert.Equal(t, "epoch 1.5/3", eta.Progress)
}

func TestComputeETA_UsesTrailingWindow(t *testing.T) {
	// Early points are slow (10 steps over the first 100s), recent points fast
	// (100 steps/10s). The window should reflect the recent rate, not the whole
	// run's average. Build 12 points; only the last etaWindowPoints (10) count.
	var pts []ml.Metric
	pts = append(pts, metric(0, 0, 0), metric(100_000, 10, 0.1))
	// Ten fast points: +100 steps every 10s starting at step 10, t=100s.
	for i := int64(1); i <= 10; i++ {
		pts = append(pts, metric(100_000+i*10_000, 10+i*100, 0.1+float64(i)*0.1))
	}
	eta := computeETA(map[string]string{"max_steps": "2000"}, pts)
	require.NotNil(t, eta)
	// Windowed rate is 10 steps/s; current step 1010, so 990 remain => 99s.
	// (The whole-run average would be far slower and give a larger estimate.)
	assert.Equal(t, int64(99), eta.RemainingSeconds)
	assert.Equal(t, "step 1010/2000", eta.Progress)
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

func TestTrainingETADisplay(t *testing.T) {
	eta := &trainingETA{RemainingSeconds: 2900, Progress: "step 5000/10000"}
	assert.Equal(t, "~48m 20s · step 5000/10000", eta.detailed())
	assert.Equal(t, "~48m 20s", eta.compact())
}

func TestTrimFloat(t *testing.T) {
	assert.Equal(t, "3", trimFloat(3.0))
	assert.Equal(t, "2.5", trimFloat(2.5))
	assert.Equal(t, "0.1", trimFloat(0.1))
}
