package utils

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordsDeploymentHistory(t *testing.T) {
	answer := func(v enrollment) <-chan enrollment {
		c := make(chan enrollment, 1)
		c <- v
		return c
	}
	recordingPlan := &deployplan.Plan{Features: map[string]struct{}{dstate.FeatureDeploymentHistoryShadow: {}}}
	for _, tc := range []struct {
		name           string
		configured     *bool
		env            string
		enrollment     <-chan enrollment
		stateRecording bool
		plan           *deployplan.Plan
		want           bool
	}{
		{"config on, not enrolled", new(true), "", answer(notEnrolled), false, nil, true},
		{"config off opts out of enrollment", new(false), "", answer(enrolled), true, nil, false},
		{"env on", nil, "true", nil, false, nil, true},
		{"env off opts out of enrollment", nil, "false", answer(enrolled), true, nil, false},
		{"no choice, enrolled", nil, "", answer(enrolled), false, nil, true},
		{"no choice, not enrolled, was recording", nil, "", answer(notEnrolled), true, recordingPlan, false},
		{"no choice, could not ask, was recording", nil, "", answer(enrollmentUnknown), true, nil, true},
		{"no choice, could not ask, was not recording", nil, "", answer(enrollmentUnknown), false, nil, false},
		{"no choice, could not ask, plan records", nil, "", answer(enrollmentUnknown), false, recordingPlan, true},
		{"no choice, could not ask, plan does not record", nil, "", answer(enrollmentUnknown), true, &deployplan.Plan{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{}
			b.Config.Experimental = &config.Experimental{DeploymentHistory: tc.configured}
			db := &b.DeploymentBundle.StateDB
			require.NoError(t, db.Open(t.Context(), filepath.Join(t.TempDir(), "resources.json"), dstate.ReadingFile, dstate.WithRecovery(false), dstate.OpenDmsArgs{}))
			if tc.stateRecording {
				db.Data.Features = map[string]struct{}{dstate.FeatureDeploymentHistoryShadow: {}}
			}
			ctx := env.Set(t.Context(), bundleenv.DeploymentHistoryVariable, tc.env)
			assert.Equal(t, tc.want, recordsDeploymentHistory(ctx, b, tc.enrollment, tc.plan))
		})
	}
}

func TestCheckAutoEnrollment(t *testing.T) {
	enrollments := func(v map[string]bool) any {
		return map[string]any{"enrollments": v}
	}
	apiError := func(status int, code string) any {
		return testserver.Response{
			StatusCode: status,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
			Body:       map[string]string{"error_code": code, "message": code},
		}
	}
	for _, tc := range []struct {
		name     string
		response any
		want     enrollment
	}{
		{"enrolled", enrollments(map[string]bool{dms.FeatureRecordHistoryAuto: true}), enrolled},
		{"not enrolled", enrollments(map[string]bool{dms.FeatureRecordHistoryAuto: false}), notEnrolled},
		{"feature unknown to the service", enrollments(map[string]bool{}), notEnrolled},
		{"workspace not served", apiError(http.StatusForbidden, "PERMISSION_DENIED"), notEnrolled},
		{"service predates the API", apiError(http.StatusNotFound, "ENDPOINT_NOT_FOUND"), notEnrolled},
		{"service failure", apiError(http.StatusBadRequest, "BAD_REQUEST"), enrollmentUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := testserver.New(t)
			server.Handle("GET", "/api/2.0/bundle/enrollments:check", func(testserver.Request) any {
				return tc.response
			})
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
			require.NoError(t, err)
			assert.Equal(t, tc.want, <-checkAutoEnrollment(t.Context(), w, enrollmentCheckTimeout))
		})
	}
}

func TestCheckAutoEnrollmentGivesUpAfterTheTimeout(t *testing.T) {
	server := testserver.New(t)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	server.Handle("GET", "/api/2.0/bundle/enrollments:check", func(testserver.Request) any {
		<-release
		return map[string]any{"enrollments": map[string]bool{dms.FeatureRecordHistoryAuto: true}}
	})
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	assert.Equal(t, enrollmentUnknown, <-checkAutoEnrollment(t.Context(), w, 50*time.Millisecond))
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		current string
		want    bool
	}{
		{"state newer major", "1.0.0", "0.300.0", true},
		{"state newer minor", "0.301.0", "0.300.0", true},
		{"state newer patch", "0.300.1", "0.300.0", true},
		{"same version", "0.300.0", "0.300.0", false},
		{"state older", "0.299.0", "0.300.0", false},
		// A dev build is built from main, so its version is the next release with a
		// -dev prerelease: newer than the last release, older than the release it
		// will become. Deploying with a dev build after a state written by the last
		// release is the normal case for a CLI developer and must not warn.
		{"state from last release, dev current", "0.300.0", "0.301.0-dev+abc123", false},
		// A released CLI reading a state written by a dev build of the same upcoming
		// release does warn: that build may have written fields this CLI lacks.
		{"dev state, released current", "0.301.0-dev+abc123", "0.300.0", true},
		// A prerelease sorts below its own release per semver.
		{"prerelease below release", "0.300.0-rc1", "0.300.0", false},
		{"release above prerelease", "0.300.0", "0.300.0-rc1", true},
		// Missing or malformed data must never produce a warning.
		{"empty state version", "", "0.300.0", false},
		{"empty current version", "0.300.0", "", false},
		{"malformed state version", "not-a-version", "0.300.0", false},
		{"malformed current version", "0.300.0", "not-a-version", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isNewerVersion(tt.state, tt.current))
		})
	}
}

func TestParseLastVersionID(t *testing.T) {
	tests := []struct {
		name       string
		deployment *bundledeployments.Deployment
		want       int
		wantErr    bool
	}{
		{"nil deployment", nil, 0, false},
		{"empty version", &bundledeployments.Deployment{}, 0, false},
		{"version zero", &bundledeployments.Deployment{LastVersionId: "0"}, 0, false},
		{"valid version", &bundledeployments.Deployment{LastVersionId: "7"}, 7, false},
		{"invalid version", &bundledeployments.Deployment{LastVersionId: "abc"}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLastVersionID(tt.deployment)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
