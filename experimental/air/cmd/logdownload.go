package aircmd

import (
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// resolveNodeCount returns the number of nodes a run used, derived from its
// accelerator type and count (accelerators are allocated in whole nodes). It
// errors when the run carries no AI runtime compute config.
func resolveNodeCount(run *jobs.Run) (int, error) {
	accelType, count := jobCompute(run)
	if accelType == "" || count <= 0 {
		return 0, fmt.Errorf("run %d has no AI runtime compute config", run.RunId)
	}
	g, err := parseGPUType(accelType)
	if err != nil {
		return 0, err
	}
	perNode, err := gpusPerNode(g)
	if err != nil {
		return 0, err
	}
	return count / perNode, nil
}
