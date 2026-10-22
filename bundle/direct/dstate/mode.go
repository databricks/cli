package dstate

import "fmt"

// Mode selects the source of resource state and the persistence required for writes.
type Mode int

const (
	// ReadingFile reads resources.json and optionally recovers a leftover WAL.
	// State writes are disabled.
	ReadingFile Mode = iota

	// WritingFile records state changes in WAL.
	// Finalize merges them into resources.json. DMS is unused.
	WritingFile

	// ReadingDMS reads resource state from DMS.
	// State writes are disabled.
	ReadingDMS

	// WritingDMS records resource state in DMS without creating a WAL.
	// DMS errors fail the deployment.
	WritingDMS

	// WritingFileWithHistory writes state through WAL to resources.json and records DMS history.
	// DMS errors and drain timeouts let the deployment continue.
	WritingFileWithHistory

	// WritingFileWithHistoryEnforced writes state through WAL to resources.json and records DMS history.
	// DMS errors and drain timeouts let the deployment continue.
	// Persists deployment_history_best_effort to reject clients that cannot honor history recording.
	WritingFileWithHistoryEnforced
)

func (m Mode) isWritable() bool {
	switch m {
	case WritingFile, WritingDMS, WritingFileWithHistory, WritingFileWithHistoryEnforced:
		return true
	default:
		return false
	}
}

func (m Mode) validate() error {
	switch m {
	case ReadingFile, WritingFile, ReadingDMS, WritingDMS, WritingFileWithHistory, WritingFileWithHistoryEnforced:
		return nil
	default:
		return fmt.Errorf("invalid deployment state mode: %d", m)
	}
}
