package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/require"
)

// Request fault mode checks that the CLI and the tests survive a single backend error: each
// planned test gets one request failed (see testserver.SetRequestFault), and the test must
// still pass. tools/fault_sweep.py drives it: it learns each test's request count from runs
// with only -faultlog, then picks a request and a fault kind per test.

// FaultPlanPath is the -faultplan flag: a JSON object mapping a test name (as go test prints it)
// to the fault to inject, e.g. {"TestAccept/bundle/deploy/basic": {"at": 3, "kind": "503"}}.
// Tests not in the plan are skipped.
var FaultPlanPath string

// FaultLogPath is the -faultlog flag: a file each test run against the testserver appends a
// FaultLogEntry line to. Setting it alone runs the tests unchanged and records request counts.
var FaultLogPath string

// PlannedFault is one -faultplan entry.
type PlannedFault struct {
	At   int    `json:"at"`
	Kind string `json:"kind"`
}

// FaultLogEntry is one line of the -faultlog file.
type FaultLogEntry struct {
	Test     string                        `json:"test"`
	Requests int                           `json:"requests"`
	Planned  *PlannedFault                 `json:"planned,omitempty"`
	Fired    *testserver.FiredRequestFault `json:"fired,omitempty"`
	Failed   bool                          `json:"failed"`
	Skipped  bool                          `json:"skipped"`
}

var (
	faultPlan    map[string]PlannedFault
	faultLogLock sync.Mutex
)

// LoadFaultPlan reads -faultplan, if set. It must run before any test starts.
func LoadFaultPlan() error {
	if FaultPlanPath == "" {
		return nil
	}
	data, err := os.ReadFile(FaultPlanPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &faultPlan); err != nil {
		return fmt.Errorf("%s: %w", FaultPlanPath, err)
	}
	if faultPlan == nil {
		faultPlan = map[string]PlannedFault{}
	}
	return nil
}

// FaultMode reports whether -faultplan or -faultlog is set. In this mode every test gets its
// own testserver, so its requests can be counted and failed.
func FaultMode() bool {
	return FaultPlanPath != "" || FaultLogPath != ""
}

// FaultPlanned returns the fault planned for the test and whether -faultplan lists it.
func FaultPlanned(t *testing.T) (PlannedFault, bool) {
	fault, ok := faultPlan[t.Name()]
	return fault, ok
}

// FaultPlanSkipReason returns why the test is skipped under -faultplan, or "".
func FaultPlanSkipReason(t *testing.T) string {
	if FaultPlanPath == "" {
		return ""
	}
	if _, ok := FaultPlanned(t); ok {
		return ""
	}
	return "not in -faultplan"
}

// setupRequestFault arms the test's planned fault on s and logs the outcome when the test ends.
// It returns the planned fault, or nil.
func setupRequestFault(t *testing.T, s *testserver.Server) *PlannedFault {
	var planned *PlannedFault
	if fault, ok := FaultPlanned(t); ok {
		require.NoError(t, s.SetRequestFault(fault.At, fault.Kind))
		planned = &fault
	}
	if FaultLogPath != "" {
		t.Cleanup(func() {
			appendFaultLog(t, FaultLogEntry{
				Test:     t.Name(),
				Requests: s.RequestCount(),
				Planned:  planned,
				Fired:    s.FiredRequestFault(),
				Failed:   t.Failed(),
				Skipped:  t.Skipped(),
			})
		})
	}
	if planned != nil {
		t.Cleanup(func() {
			if fired := s.FiredRequestFault(); fired != nil {
				t.Logf("Injected fault %s into request %d: %s %s", fired.Kind, fired.At, fired.Method, fired.Path)
			} else {
				t.Logf("Planned fault %s at request %d did not fire: the test made %d requests", planned.Kind, planned.At, s.RequestCount())
			}
		})
	}
	return planned
}

func appendFaultLog(t *testing.T, entry FaultLogEntry) {
	line, err := json.Marshal(entry)
	require.NoError(t, err)

	faultLogLock.Lock()
	defer faultLogLock.Unlock()
	f, err := os.OpenFile(FaultLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	require.NoError(t, err)
}
