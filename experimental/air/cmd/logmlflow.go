package aircmd

import (
	"context"
	"fmt"
	"io"

	"github.com/databricks/databricks-sdk-go"
)

// mlflowLogFallback prints a run's logs from MLflow artifacts. It is the fallback
// used when Bricklens can't serve logs (errBricklensFeatureDisabled). Both the
// completed-run tail (--lines) and the active-run stream, plus the --minutes
// window, are honored here via req.
//
// TODO(air-logs-m3): port print_all_logs / monitor_and_stream_logs — MLflow
// artifact discovery (logs/[attempt_X/]node_Y), chunk walking (logs-N.chunk.txt),
// and credential-vending download. Until then the fallback reports that the
// legacy path is not yet available so the failure is explicit rather than silent.
func mlflowLogFallback(ctx context.Context, w *databricks.WorkspaceClient, out io.Writer, req logRequest, status logRunStatus) (bool, error) {
	return false, fmt.Errorf("logs for run %d are not available via Bricklens and the MLflow fallback is not yet implemented", req.runID)
}
