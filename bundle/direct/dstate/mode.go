package dstate

import "fmt"

// Mode selects the source of resource state and the persistence required for writes.
type Mode int

const (
	ReadingFile Mode = iota
	WritingFile
	ReadingDMS
	WritingDMS
	WritingFileWithHistory
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

func (m Mode) usesDMSState() bool {
	return m == ReadingDMS || m == WritingDMS
}

func (m Mode) isFileWithHistory() bool {
	return m == WritingFileWithHistory || m == WritingFileWithHistoryEnforced
}

func (m Mode) recordsHistory() bool {
	return m.usesDMSState() || m.isFileWithHistory()
}

func (m Mode) validate() error {
	switch m {
	case ReadingFile, WritingFile, ReadingDMS, WritingDMS, WritingFileWithHistory, WritingFileWithHistoryEnforced:
		return nil
	default:
		return fmt.Errorf("invalid deployment state mode: %d", m)
	}
}
