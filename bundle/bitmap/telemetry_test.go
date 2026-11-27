package bitmap

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBitsRejectsUnsupportedTelemetryField(t *testing.T) {
	_, err := NewBuilder(EmbeddedSchema()).Bits(struct {
		Count int `metric:"count"`
	}{})
	assert.ErrorContains(t, err, "telemetry field Count has unsupported type int")
}

// A field without a metric tag is named by its Go name in snake case.
func TestTelemetryPathsWithoutMetricTag(t *testing.T) {
	paths, err := telemetryPaths(reflect.TypeFor[struct {
		Tagged         bool `metric:"legacy_key"`
		DMSCompatAuto  bool
		AppStarted     *bool
		SQLWarehouseOn BoolSetForTest
	}]())
	require.NoError(t, err)
	assert.Equal(t, []string{
		"telemetry.legacy_key",
		"telemetry.dms_compat_auto",
		"telemetry.app_started", "telemetry.app_started=true", "telemetry.app_started=false",
		"telemetry.sql_warehouse_on", "telemetry.sql_warehouse_on=true", "telemetry.sql_warehouse_on=false",
	}, paths)
}

func TestTelemetryPathsRejectsDuplicateNames(t *testing.T) {
	_, err := telemetryPaths(reflect.TypeFor[struct {
		SelectUsed bool
		Other      bool `metric:"select_used"`
	}]())
	assert.ErrorContains(t, err, "telemetry fields SelectUsed and Other are both named telemetry.select_used")
}

// BoolSetForTest stands in for bundle.BoolSet, which this package cannot import.
type BoolSetForTest struct{}

func (BoolSetForTest) Values() []bool { return nil }
