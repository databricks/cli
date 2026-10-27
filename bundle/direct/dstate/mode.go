package dstate

// Mode selects the source of resource state and the persistence required for writes.
type Mode int

const (
	// ReadingFile reads resources.json and optionally recovers a leftover WAL.
	// State writes are disabled.
	ReadingFile Mode = iota

	// WritingFile records state changes in WAL.
	// Finalize merges them into resources.json. DMS is unused.
	WritingFile

	// WritingDMS reads and writes resource state through DMS without creating a WAL.
	// DMS errors fail the deployment.
	WritingDMS

	// WritingDeploymentHistoryShadow writes state through WAL to resources.json and records DMS history.
	// DMS errors and drain timeouts let the deployment continue.
	WritingDeploymentHistoryShadow

	// WritingDeploymentHistoryShadowEnforced writes state through WAL to resources.json and records DMS history.
	// DMS errors and drain timeouts let the deployment continue.
	// Persists deployment_history_shadow to reject clients that cannot honor history recording.
	WritingDeploymentHistoryShadowEnforced
)

func (m Mode) isWritable() bool {
	switch m {
	case WritingFile, WritingDMS, WritingDeploymentHistoryShadow, WritingDeploymentHistoryShadowEnforced:
		return true
	default:
		return false
	}
}
