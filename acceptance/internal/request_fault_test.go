package internal

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// planFault plans a fault for the running test, as -faultplan would.
func planFault(t *testing.T, fault PlannedFault) {
	oldPath, oldPlan := FaultPlanPath, faultPlan
	t.Cleanup(func() { FaultPlanPath, faultPlan = oldPath, oldPlan })
	FaultPlanPath = "plan.json"
	faultPlan = map[string]PlannedFault{t.Name(): fault}
}

func get(t *testing.T, url string) int {
	resp, err := http.Get(url)
	require.NoError(t, err)
	resp.Body.Close()
	return resp.StatusCode
}

func TestRequestFaultDropsRetriedAttempt(t *testing.T) {
	planFault(t, PlannedFault{At: 2, Kind: testserver.FaultUnavailable})
	outputDir := t.TempDir()
	stubs := []ServerStub{
		{Pattern: "GET /api/2.0/ok", Response: testserver.Response{Body: "{}"}},
		// A stub answering with an error keeps every attempt recorded, fault or not.
		{Pattern: "GET /api/2.0/stub-error", Response: testserver.Response{StatusCode: 503, Body: "{}"}},
	}
	url := startLocalServer(t, stubs, true, false, nil, outputDir)

	assert.Equal(t, 200, get(t, url+"/api/2.0/ok"))
	assert.Equal(t, 503, get(t, url+"/api/2.0/ok"))
	assert.Equal(t, 200, get(t, url+"/api/2.0/ok"))
	assert.Equal(t, 503, get(t, url+"/api/2.0/stub-error"))
	assert.Equal(t, 503, get(t, url+"/api/2.0/stub-error"))

	data, err := os.ReadFile(filepath.Join(outputDir, "out.requests.txt"))
	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(string(data), `"path": "/api/2.0/ok"`))
	assert.Equal(t, 2, strings.Count(string(data), `"path": "/api/2.0/stub-error"`))
}

func TestFaultPlanSkipReason(t *testing.T) {
	assert.Empty(t, FaultPlanSkipReason(t))

	planFault(t, PlannedFault{At: 1, Kind: testserver.FaultUnavailable})
	assert.Empty(t, FaultPlanSkipReason(t))

	faultPlan = map[string]PlannedFault{"TestAccept/other": {At: 1, Kind: testserver.FaultUnavailable}}
	assert.Equal(t, "not in -faultplan", FaultPlanSkipReason(t))
}
