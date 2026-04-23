package ucm

import (
	"github.com/databricks/cli/libs/telemetry/protos"
)

// Metrics collects telemetry-safe measurements from across the ucm codebase.
// Mirrors bundle.Metrics; starts minimal and will grow as more telemetry lands.
type Metrics struct {
	ExecutionTimes []protos.IntMapEntry
}
