package dagger

import (
	"log/slog"
	"sync/atomic"
	"time"
)

// DaggerMetrics tracks usage statistics for Dagger sandboxes.
type DaggerMetrics struct {
	ValidationCount   atomic.Int64
	SuccessCount      atomic.Int64
	FallbackCount     atomic.Int64
	TotalDurationMs   atomic.Int64
	validationCount   int64
}

// GlobalMetrics provides global metrics tracking for all Dagger operations.
var GlobalMetrics = &DaggerMetrics{}

// RecordValidation records metrics for a validation operation.
// This should be called after each validation completes.
func RecordValidation(logger *slog.Logger, success bool, duration time.Duration) {
	GlobalMetrics.ValidationCount.Add(1)
	GlobalMetrics.TotalDurationMs.Add(duration.Milliseconds())

	if success {
		GlobalMetrics.SuccessCount.Add(1)
	}

	count := GlobalMetrics.ValidationCount.Load()
	avgDuration := float64(GlobalMetrics.TotalDurationMs.Load()) / float64(count)

	logger.Info("Validation completed",
		"success", success,
		"duration_ms", duration.Milliseconds(),
		"sandbox", "dagger",
		"total_validations", count,
		"avg_duration_ms", avgDuration)
}

// RecordFallback records when Dagger fails and falls back to local sandbox.
func RecordFallback(logger *slog.Logger, reason string) {
	GlobalMetrics.FallbackCount.Add(1)

	logger.Warn("Dagger fallback to local sandbox",
		"reason", reason,
		"fallback_count", GlobalMetrics.FallbackCount.Load())
}

// GetMetrics returns a snapshot of current metrics.
func GetMetrics() map[string]interface{} {
	validations := GlobalMetrics.ValidationCount.Load()
	totalDuration := GlobalMetrics.TotalDurationMs.Load()

	avgDuration := float64(0)
	if validations > 0 {
		avgDuration = float64(totalDuration) / float64(validations)
	}

	return map[string]interface{}{
		"validation_count":    validations,
		"success_count":       GlobalMetrics.SuccessCount.Load(),
		"fallback_count":      GlobalMetrics.FallbackCount.Load(),
		"avg_duration_ms":     avgDuration,
		"total_duration_ms":   totalDuration,
	}
}
