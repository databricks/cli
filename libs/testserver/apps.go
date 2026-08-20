package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

const (
	appStatusRunningMessage     = "App has status: App is running"
	appStatusUnavailableMessage = "App status is unavailable."
)

func (s *FakeWorkspace) AppsCreateUpdate(req Request, name string) Response {
	var updateReq apps.AsyncUpdateAppRequest
	if err := json.Unmarshal(req.Body, &updateReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	defer s.LockUnlock()()

	existing, ok := s.Apps[name]
	if !ok {
		return Response{StatusCode: 404}
	}

	if updateReq.App != nil {
		// Convert both to maps and apply only the fields listed in update_mask.
		existingJSON, err := json.Marshal(existing)
		if err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}
		var existingMap map[string]any
		if err := json.Unmarshal(existingJSON, &existingMap); err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}

		updateJSON, err := json.Marshal(updateReq.App)
		if err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}
		var updateMap map[string]any
		if err := json.Unmarshal(updateJSON, &updateMap); err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}

		for field := range strings.SplitSeq(updateReq.UpdateMask, ",") {
			if v, ok := updateMap[strings.TrimSpace(field)]; ok {
				existingMap[strings.TrimSpace(field)] = v
			}
		}

		merged, err := json.Marshal(existingMap)
		if err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}
		if err := json.Unmarshal(merged, &existing); err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}
	}
	s.Apps[name] = existing

	return Response{
		Body: apps.AppUpdate{
			Status: &apps.AppUpdateUpdateStatus{
				State: apps.AppUpdateUpdateStatusUpdateStateSucceeded,
			},
		},
	}
}

func (s *FakeWorkspace) AppsGetUpdate(_ Request, name string) Response {
	defer s.LockUnlock()()

	_, ok := s.Apps[name]
	if !ok {
		return Response{StatusCode: 404}
	}

	return Response{
		Body: apps.AppUpdate{
			Status: &apps.AppUpdateUpdateStatus{
				State: apps.AppUpdateUpdateStatusUpdateStateSucceeded,
			},
		},
	}
}

func (s *FakeWorkspace) AppsCreateDeployment(req Request, name string) Response {
	defer s.LockUnlock()()

	app, ok := s.Apps[name]
	if !ok {
		return Response{StatusCode: 404}
	}

	var deployment apps.AppDeployment
	if err := json.Unmarshal(req.Body, &deployment); err != nil {
		return Response{StatusCode: 500, Body: fmt.Sprintf("internal error: %s", err)}
	}

	deployment.DeploymentId = fmt.Sprintf("deploy-%d", nextID())
	deployment.Status = &apps.AppDeploymentStatus{
		State:   apps.AppDeploymentStateSucceeded,
		Message: "Deployment succeeded",
	}

	app.ActiveDeployment = &deployment
	app.DefaultSourceCodePath = deployment.SourceCodePath
	s.Apps[name] = app

	return Response{Body: deployment}
}

func (s *FakeWorkspace) AppsGetDeployment(_ Request, name, deploymentID string) Response {
	defer s.LockUnlock()()

	app, ok := s.Apps[name]
	if !ok {
		return Response{StatusCode: 404}
	}

	if app.ActiveDeployment == nil || app.ActiveDeployment.DeploymentId != deploymentID {
		return Response{StatusCode: 404}
	}

	// Return a copy so the masking below does not mutate the stored deployment.
	deployment := *app.ActiveDeployment

	// The real GET response strips env var values, returning only the name of
	// each variable. Reproduce that masking so the local golden matches
	// recorded cloud behavior.
	maskedEnvVars := make([]apps.EnvVar, len(deployment.EnvVars))
	for i, ev := range deployment.EnvVars {
		maskedEnvVars[i] = apps.EnvVar{Name: ev.Name}
	}
	deployment.EnvVars = maskedEnvVars

	return Response{Body: deployment}
}

func (s *FakeWorkspace) AppsStart(_ Request, name string) Response {
	defer s.LockUnlock()()

	app, ok := s.Apps[name]
	if !ok {
		return Response{StatusCode: 404}
	}

	app.ComputeStatus = &apps.ComputeStatus{
		State:   apps.ComputeStateActive,
		Message: "App compute is active.",
	}
	// Starting the compute brings the application up.
	app.AppStatus = &apps.ApplicationStatus{
		State:   "RUNNING",
		Message: appStatusRunningMessage,
	}
	s.Apps[name] = app

	return Response{Body: app}
}

func (s *FakeWorkspace) AppsStop(_ Request, name string) Response {
	defer s.LockUnlock()()

	app, ok := s.Apps[name]
	if !ok {
		return Response{StatusCode: 404}
	}

	app.ComputeStatus = &apps.ComputeStatus{
		State:   apps.ComputeStateStopped,
		Message: "Start the app compute to deploy the app.",
	}
	// Stopping the compute takes the application down.
	app.AppStatus = &apps.ApplicationStatus{
		State:   "UNAVAILABLE",
		Message: appStatusUnavailableMessage,
	}
	// The backend clears both deployments on stop for the apps these fixtures use,
	// so the deploy-only fields read back empty. Match that so drift tests are realistic.
	app.ActiveDeployment = nil
	app.PendingDeployment = nil
	s.Apps[name] = app

	return Response{Body: app}
}

// AppsGet returns the app, keeping DELETING resources visible so callers can
// observe transient state (matches the cloud DELETE lifecycle).
func (s *FakeWorkspace) AppsGet(name string) Response {
	defer s.LockUnlock()()

	app, ok := s.Apps[name]
	if !ok {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Resource apps.App not found: %v", name)},
		}
	}

	return Response{Body: app}
}

// AppsDelete simulates the real Apps DELETE lifecycle: the first DELETE flips
// the app into DELETING state (without removing it), and a second DELETE while
// still in DELETING returns 400 with the exact cloud error message.
func (s *FakeWorkspace) AppsDelete(name string) Response {
	defer s.LockUnlock()()

	app, ok := s.Apps[name]
	if !ok {
		return Response{StatusCode: 404}
	}

	if app.ComputeStatus != nil && app.ComputeStatus.State == apps.ComputeStateDeleting {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "BAD_REQUEST",
				"message": fmt.Sprintf(
					"Cannot delete app %s as it is not terminal with state DELETING, "+
						"and was updated less than 20 minutes ago. Please wait before trying again.", name),
			},
		}
	}

	app.ComputeStatus = &apps.ComputeStatus{
		State:   apps.ComputeStateDeleting,
		Message: "App is being deleted.",
	}
	app.AppStatus = &apps.ApplicationStatus{
		State:   "UNAVAILABLE",
		Message: appStatusUnavailableMessage,
	}
	s.Apps[name] = app

	return Response{}
}

func (s *FakeWorkspace) AppsUpsert(req Request, name string) Response {
	var app apps.App

	if err := json.Unmarshal(req.Body, &app); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	defer s.LockUnlock()()

	if name != "" {
		_, ok := s.Apps[name]
		if !ok {
			return Response{
				StatusCode: 404,
			}
		}
	} else {
		name = app.Name
		if name == "" {
			return Response{
				StatusCode: 400,
				Body:       "name is required",
			}
		}
		// Check if app already exists on create
		if _, exists := s.Apps[name]; exists {
			return Response{
				StatusCode: 409,
				Body: map[string]string{
					"error_code": "RESOURCE_ALREADY_EXISTS",
					"message":    "An app with the same name already exists: " + name,
				},
			}
		}
	}

	// A no_compute app is created without running compute, so on cloud it
	// reports an UNAVAILABLE application status until it is started.
	if req.URL.Query().Get("no_compute") == "true" {
		app.AppStatus = &apps.ApplicationStatus{
			State:   "UNAVAILABLE",
			Message: appStatusUnavailableMessage,
		}
		app.ComputeStatus = &apps.ComputeStatus{
			State:   apps.ComputeStateStopped,
			Message: "App compute is stopped.",
		}
	} else {
		app.AppStatus = &apps.ApplicationStatus{
			State:   "RUNNING",
			Message: "Application is running.",
		}
		app.ComputeStatus = &apps.ComputeStatus{
			State:   "ACTIVE",
			Message: "App compute is active.",
		}

		// Simulate the apps platform side effect: when an app is created, it is deployed with the default source code path.
		deployment := apps.AppDeployment{
			SourceCodePath: "/Workspace/Users/tester@databricks.com/" + name,
		}

		deployment.DeploymentId = fmt.Sprintf("deploy-%d", nextID())
		deployment.Status = &apps.AppDeploymentStatus{
			State:   apps.AppDeploymentStateSucceeded,
			Message: "Deployment succeeded",
		}

		app.ActiveDeployment = &deployment
		app.DefaultSourceCodePath = deployment.SourceCodePath
	}

	app.Url = name + "-123.cloud.databricksapps.com"
	app.Id = strconv.Itoa(len(s.Apps) + 1000)

	if app.ComputeSize == "" {
		app.ComputeSize = "MEDIUM"
	}

	// The platform enables user access token forwarding regardless of what the
	// request asked for, so the remote always reports true.
	app.ForwardUserAccessToken = true

	// Assign a service principal to the app, mimicking the real platform.
	if app.ServicePrincipalClientId == "" {
		app.ServicePrincipalClientId = nextUUID()
		app.ServicePrincipalId = nextID()
		app.ServicePrincipalName = "app-" + name
	}

	// Simulate the apps platform side effect: when an app references a job
	// with a permission, the platform grants that permission to the app's
	// service principal on the referenced resource.
	for _, res := range app.Resources {
		if res.Job == nil {
			continue
		}
		s.upsertPermission("/jobs/"+res.Job.Id, iam.AccessControlResponse{
			ServicePrincipalName: app.ServicePrincipalName,
			AllPermissions: []iam.Permission{{
				PermissionLevel: iam.PermissionLevel(res.Job.Permission),
				ForceSendFields: []string{"Inherited"},
			}},
		})
	}

	s.Apps[name] = app
	return Response{
		Body: app,
	}
}
