package dagger

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/databricks/cli/libs/log"
)

// DaggerMetrics tracks usage statistics for Dagger sandboxes.
type DaggerMetrics struct {
	ValidationCount atomic.Int64
	SuccessCount    atomic.Int64
	FallbackCount   atomic.Int64
	TotalDurationMs atomic.Int64
}

// GlobalMetrics provides global metrics tracking for all Dagger operations.
var GlobalMetrics = &DaggerMetrics{}

// RecordValidation records metrics for a validation operation.
// This should be called after each validation completes.
func RecordValidation(ctx context.Context, success bool, duration time.Duration) {
	GlobalMetrics.ValidationCount.Add(1)
	GlobalMetrics.TotalDurationMs.Add(duration.Milliseconds())

	if success {
		GlobalMetrics.SuccessCount.Add(1)
	}

	count := GlobalMetrics.ValidationCount.Load()
	avgDuration := float64(GlobalMetrics.TotalDurationMs.Load()) / float64(count)

	log.Infof(ctx, "Validation completed (success: %v, duration_ms: %d, sandbox: dagger, total_validations: %d, avg_duration_ms: %.2f)",
		success, duration.Milliseconds(), count, avgDuration)
}

// RecordFallback records when Dagger fails and falls back to local sandbox.
func RecordFallback(ctx context.Context, reason string) {
	GlobalMetrics.FallbackCount.Add(1)

	log.Warnf(ctx, "Dagger fallback to local sandbox (reason: %s, fallback_count: %d)",
		reason, GlobalMetrics.FallbackCount.Load())
}

// GetMetrics returns a snapshot of current metrics.
func GetMetrics() map[string]any {
	validations := GlobalMetrics.ValidationCount.Load()
	totalDuration := GlobalMetrics.TotalDurationMs.Load()

	avgDuration := float64(0)
	if validations > 0 {
		avgDuration = float64(totalDuration) / float64(validations)
	}

	return map[string]any{
		"validation_count":  validations,
		"success_count":     GlobalMetrics.SuccessCount.Load(),
		"fallback_count":    GlobalMetrics.FallbackCount.Load(),
		"avg_duration_ms":   avgDuration,
		"total_duration_ms": totalDuration,
	}
}
