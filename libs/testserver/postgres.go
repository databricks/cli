package testserver

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/common/types/duration"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

func nowTime() *sdktime.Time {
	return sdktime.New(time.Now().UTC())
}

// postgresErrorResponse creates an error response with trace ID for postgres API.
// The trace ID is included directly in the message to match the real API behavior.
func postgresErrorResponse(statusCode int, message string) Response {
	traceID := fmt.Sprintf("%x", nextID())
	return Response{
		StatusCode: statusCode,
		Body: map[string]string{
			"message": fmt.Sprintf("%s [TraceId: %s]", message, traceID),
		},
	}
}

// PostgresProjectCreate creates a new postgres project.
func (s *FakeWorkspace) PostgresProjectCreate(req Request, projectID string) Response {
	defer s.LockUnlock()()

	var project postgres.Project
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &project); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			}
		}
	}

	name := "projects/" + projectID

	// Check for duplicate
	if _, exists := s.PostgresProjects[name]; exists {
		return postgresErrorResponse(409, "project with such id already exists in the workspace")
	}

	now := nowTime()
	project.Name = name
	project.Uid = nextUUID()
	project.CreateTime = now
	project.UpdateTime = now

	// Copy spec fields to status (API returns status as materialized view)
	if project.Spec != nil {
		project.Status = &postgres.ProjectStatus{
			DisplayName:                project.Spec.DisplayName,
			PgVersion:                  project.Spec.PgVersion,
			HistoryRetentionDuration:   project.Spec.HistoryRetentionDuration,
			Owner:                      TestUser.UserName,
			BranchLogicalSizeLimitBytes: 1073741824, // 1 GB default
			SyntheticStorageSizeBytes:   0,
			ForceSendFields:             []string{"SyntheticStorageSizeBytes"},
		}
		if project.Spec.DefaultEndpointSettings != nil {
			project.Status.DefaultEndpointSettings = &postgres.ProjectDefaultEndpointSettings{
				AutoscalingLimitMinCu:  project.Spec.DefaultEndpointSettings.AutoscalingLimitMinCu,
				AutoscalingLimitMaxCu:  project.Spec.DefaultEndpointSettings.AutoscalingLimitMaxCu,
				SuspendTimeoutDuration: project.Spec.DefaultEndpointSettings.SuspendTimeoutDuration,
			}
		}
		// Clear spec so it's not returned in response (API only returns status)
		project.Spec = nil
	}

	s.PostgresProjects[name] = project

	// Create a default branch for the project
	s.createDefaultBranchLocked(name)

	return Response{
		Body: s.createOperationLocked(project.Name, project),
	}
}

// PostgresProjectGet retrieves a postgres project by name.
func (s *FakeWorkspace) PostgresProjectGet(name string) Response {
	defer s.LockUnlock()()

	project, exists := s.PostgresProjects[name]
	if !exists {
		return postgresErrorResponse(404, "project id not found")
	}

	return Response{
		Body: project,
	}
}

// PostgresProjectList lists all postgres projects.
func (s *FakeWorkspace) PostgresProjectList() Response {
	defer s.LockUnlock()()

	projects := make([]postgres.Project, 0, len(s.PostgresProjects))
	for _, p := range s.PostgresProjects {
		projects = append(projects, p)
	}

	return Response{
		Body: postgres.ListProjectsResponse{
			Projects: projects,
		},
	}
}

// PostgresProjectUpdate updates a postgres project.
func (s *FakeWorkspace) PostgresProjectUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	project, exists := s.PostgresProjects[name]
	if !exists {
		return postgresErrorResponse(404, "project id not found")
	}

	var updateProject postgres.Project
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &updateProject); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			}
		}
	}

	// Apply updates from spec to status
	if updateProject.Spec != nil {
		if project.Status == nil {
			project.Status = &postgres.ProjectStatus{}
		}
		if updateProject.Spec.DisplayName != "" {
			project.Status.DisplayName = updateProject.Spec.DisplayName
		}
		if updateProject.Spec.DefaultEndpointSettings != nil {
			if project.Status.DefaultEndpointSettings == nil {
				project.Status.DefaultEndpointSettings = &postgres.ProjectDefaultEndpointSettings{}
			}
			if updateProject.Spec.DefaultEndpointSettings.AutoscalingLimitMinCu != 0 {
				project.Status.DefaultEndpointSettings.AutoscalingLimitMinCu = updateProject.Spec.DefaultEndpointSettings.AutoscalingLimitMinCu
			}
			if updateProject.Spec.DefaultEndpointSettings.AutoscalingLimitMaxCu != 0 {
				project.Status.DefaultEndpointSettings.AutoscalingLimitMaxCu = updateProject.Spec.DefaultEndpointSettings.AutoscalingLimitMaxCu
			}
		}
	}

	project.UpdateTime = nowTime()
	s.PostgresProjects[name] = project

	return Response{
		Body: s.createOperationLocked(project.Name, project),
	}
}

// PostgresProjectDelete deletes a postgres project.
func (s *FakeWorkspace) PostgresProjectDelete(name string) Response {
	defer s.LockUnlock()()

	if _, exists := s.PostgresProjects[name]; !exists {
		return postgresErrorResponse(404, "project id not found")
	}

	delete(s.PostgresProjects, name)

	return Response{
		Body: s.createOperationLocked(name, nil),
	}
}

// PostgresBranchCreate creates a new postgres branch.
func (s *FakeWorkspace) PostgresBranchCreate(req Request, parent, branchID string) Response {
	defer s.LockUnlock()()

	// Check if parent project exists
	if _, exists := s.PostgresProjects[parent]; !exists {
		return postgresErrorResponse(404, "project id not found")
	}

	var branch postgres.Branch
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &branch); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			}
		}
	}

	name := fmt.Sprintf("%s/branches/%s", parent, branchID)

	// Check for duplicate
	if _, exists := s.PostgresBranches[name]; exists {
		return postgresErrorResponse(409, "branch with such id already exists")
	}

	now := nowTime()
	branch.Name = name
	branch.Parent = parent
	branch.Uid = "br-" + nextUUID()[:20]
	branch.CreateTime = now
	branch.UpdateTime = now

	// Find the default branch to use as source
	var defaultBranch *postgres.Branch
	prefix := parent + "/branches/"
	for brName, br := range s.PostgresBranches {
		if strings.HasPrefix(brName, prefix) && br.Status != nil && br.Status.Default {
			defaultBranch = &br
			break
		}
	}

	// Initialize status with all required fields
	branch.Status = &postgres.BranchStatus{
		CurrentState:     postgres.BranchStatusStateReady,
		StateChangeTime:  now,
		Default:          false,
		IsProtected:      false,
		LogicalSizeBytes: 0,
		ForceSendFields:  []string{"Default", "IsProtected", "LogicalSizeBytes"},
	}

	// Set source branch info if a default branch exists
	if defaultBranch != nil {
		branch.Status.SourceBranch = defaultBranch.Name
		branch.Status.SourceBranchLsn = "0/0"
		branch.Status.SourceBranchTime = nowTime()
	}

	// Clear spec - API only returns status
	branch.Spec = nil

	s.PostgresBranches[name] = branch

	return Response{
		Body: s.createOperationLocked(branch.Name, branch),
	}
}

// PostgresBranchGet retrieves a postgres branch by name.
func (s *FakeWorkspace) PostgresBranchGet(name string) Response {
	defer s.LockUnlock()()

	// Extract project name from branch name (format: projects/{project}/branches/{branch})
	parts := strings.Split(name, "/branches/")
	if len(parts) == 2 {
		projectName := parts[0]
		if _, exists := s.PostgresProjects[projectName]; !exists {
			return postgresErrorResponse(404, "project id not found")
		}
	}

	branch, exists := s.PostgresBranches[name]
	if !exists {
		return postgresErrorResponse(404, "branch id not found")
	}

	return Response{
		Body: branch,
	}
}

// PostgresBranchList lists all postgres branches for a project.
func (s *FakeWorkspace) PostgresBranchList(parent string) Response {
	defer s.LockUnlock()()

	// Check if parent project exists
	if _, exists := s.PostgresProjects[parent]; !exists {
		return postgresErrorResponse(404, "project id not found")
	}

	var branches []postgres.Branch
	prefix := parent + "/branches/"
	for name, b := range s.PostgresBranches {
		if strings.HasPrefix(name, prefix) {
			branches = append(branches, b)
		}
	}

	return Response{
		Body: postgres.ListBranchesResponse{
			Branches: branches,
		},
	}
}

// PostgresBranchUpdate updates a postgres branch.
func (s *FakeWorkspace) PostgresBranchUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	branch, exists := s.PostgresBranches[name]
	if !exists {
		return postgresErrorResponse(404, "branch id not found")
	}

	var updateBranch postgres.Branch
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &updateBranch); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			}
		}
	}

	// Apply updates from spec to status
	if updateBranch.Spec != nil {
		if branch.Status == nil {
			branch.Status = &postgres.BranchStatus{}
		}
		branch.Status.IsProtected = updateBranch.Spec.IsProtected
	}

	branch.UpdateTime = nowTime()
	s.PostgresBranches[name] = branch

	return Response{
		Body: s.createOperationLocked(branch.Name, branch),
	}
}

// PostgresBranchDelete deletes a postgres branch.
func (s *FakeWorkspace) PostgresBranchDelete(name string) Response {
	defer s.LockUnlock()()

	branch, exists := s.PostgresBranches[name]
	if !exists {
		return postgresErrorResponse(404, "branch id not found")
	}

	// Check if branch is protected
	if branch.Status != nil && branch.Status.IsProtected {
		return postgresErrorResponse(400, "cannot delete protected branch")
	}

	delete(s.PostgresBranches, name)

	return Response{
		Body: s.createOperationLocked(name, nil),
	}
}

// PostgresEndpointCreate creates a new postgres endpoint.
func (s *FakeWorkspace) PostgresEndpointCreate(req Request, parent, endpointID string) Response {
	defer s.LockUnlock()()

	// Check if parent branch exists
	branch, exists := s.PostgresBranches[parent]
	if !exists {
		return postgresErrorResponse(404, "branch id not found")
	}

	var endpoint postgres.Endpoint
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &endpoint); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			}
		}
	}

	name := fmt.Sprintf("%s/endpoints/%s", parent, endpointID)

	// Check for duplicate
	if _, exists := s.PostgresEndpoints[name]; exists {
		return postgresErrorResponse(409, "endpoint with such id already exists")
	}

	now := nowTime()
	endpoint.Name = name
	endpoint.Parent = parent
	endpoint.Uid = "ep-" + nextUUID()[:20]
	endpoint.CreateTime = now
	endpoint.UpdateTime = now

	// Get default suspend timeout from project
	var defaultSuspendTimeout *duration.Duration
	if project, ok := s.PostgresProjects[branch.Parent]; ok {
		if project.Status != nil && project.Status.DefaultEndpointSettings != nil {
			defaultSuspendTimeout = project.Status.DefaultEndpointSettings.SuspendTimeoutDuration
		}
	}

	// Initialize status with all required fields
	endpoint.Status = &postgres.EndpointStatus{
		CurrentState:           postgres.EndpointStatusStateActive,
		Disabled:               false,
		Settings:               &postgres.EndpointSettings{},
		SuspendTimeoutDuration: defaultSuspendTimeout,
		ForceSendFields:        []string{"Disabled"},
	}

	// Copy spec values to status
	if endpoint.Spec != nil {
		endpoint.Status.EndpointType = endpoint.Spec.EndpointType
		endpoint.Status.AutoscalingLimitMinCu = endpoint.Spec.AutoscalingLimitMinCu
		endpoint.Status.AutoscalingLimitMaxCu = endpoint.Spec.AutoscalingLimitMaxCu
		if endpoint.Spec.Disabled {
			endpoint.Status.Disabled = true
		}
	}

	// Generate a fake host
	endpoint.Status.Hosts = &postgres.EndpointHosts{
		Host: endpoint.Uid + ".database.us-east-1.cloud.databricks.com",
	}

	// Clear spec - API only returns status
	endpoint.Spec = nil

	s.PostgresEndpoints[name] = endpoint

	return Response{
		Body: s.createOperationLocked(endpoint.Name, endpoint),
	}
}

// PostgresEndpointGet retrieves a postgres endpoint by name.
func (s *FakeWorkspace) PostgresEndpointGet(name string) Response {
	defer s.LockUnlock()()

	// Extract project and branch names from endpoint name
	// Format: projects/{project}/branches/{branch}/endpoints/{endpoint}
	parts := strings.Split(name, "/branches/")
	if len(parts) == 2 {
		projectName := parts[0]
		if _, exists := s.PostgresProjects[projectName]; !exists {
			return postgresErrorResponse(404, "project id not found")
		}
		// Check if branch exists
		branchParts := strings.Split(parts[1], "/endpoints/")
		if len(branchParts) == 2 {
			branchName := projectName + "/branches/" + branchParts[0]
			if _, exists := s.PostgresBranches[branchName]; !exists {
				return postgresErrorResponse(404, "branch id not found")
			}
		}
	}

	endpoint, exists := s.PostgresEndpoints[name]
	if !exists {
		return postgresErrorResponse(404, "endpoint id not found")
	}

	return Response{
		Body: endpoint,
	}
}

// PostgresEndpointList lists all postgres endpoints for a branch.
func (s *FakeWorkspace) PostgresEndpointList(parent string) Response {
	defer s.LockUnlock()()

	// Check if parent branch exists
	if _, exists := s.PostgresBranches[parent]; !exists {
		return postgresErrorResponse(404, "branch id not found")
	}

	var endpoints []postgres.Endpoint
	prefix := parent + "/endpoints/"
	for name, e := range s.PostgresEndpoints {
		if strings.HasPrefix(name, prefix) {
			endpoints = append(endpoints, e)
		}
	}

	return Response{
		Body: postgres.ListEndpointsResponse{
			Endpoints: endpoints,
		},
	}
}

// PostgresEndpointUpdate updates a postgres endpoint.
func (s *FakeWorkspace) PostgresEndpointUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	endpoint, exists := s.PostgresEndpoints[name]
	if !exists {
		return postgresErrorResponse(404, "endpoint id not found")
	}

	var updateEndpoint postgres.Endpoint
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &updateEndpoint); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			}
		}
	}

	// Apply updates from spec to status
	if updateEndpoint.Spec != nil {
		if endpoint.Status == nil {
			endpoint.Status = &postgres.EndpointStatus{}
		}
		if updateEndpoint.Spec.AutoscalingLimitMinCu != 0 {
			endpoint.Status.AutoscalingLimitMinCu = updateEndpoint.Spec.AutoscalingLimitMinCu
		}
		if updateEndpoint.Spec.AutoscalingLimitMaxCu != 0 {
			endpoint.Status.AutoscalingLimitMaxCu = updateEndpoint.Spec.AutoscalingLimitMaxCu
		}
		endpoint.Status.Disabled = updateEndpoint.Spec.Disabled
	}

	endpoint.UpdateTime = nowTime()
	s.PostgresEndpoints[name] = endpoint

	return Response{
		Body: s.createOperationLocked(endpoint.Name, endpoint),
	}
}

// PostgresEndpointDelete deletes a postgres endpoint.
func (s *FakeWorkspace) PostgresEndpointDelete(name string) Response {
	defer s.LockUnlock()()

	if _, exists := s.PostgresEndpoints[name]; !exists {
		return postgresErrorResponse(404, "endpoint id not found")
	}

	delete(s.PostgresEndpoints, name)

	return Response{
		Body: s.createOperationLocked(name, nil),
	}
}

// PostgresOperationGet retrieves a postgres operation by name.
func (s *FakeWorkspace) PostgresOperationGet(name string) Response {
	defer s.LockUnlock()()

	operation, exists := s.PostgresOperations[name]
	if !exists {
		return postgresErrorResponse(404, "operation not found")
	}

	return Response{
		Body: operation,
	}
}

// createOperationLocked creates and stores an operation (caller must hold lock).
func (s *FakeWorkspace) createOperationLocked(resourceName string, response any) postgres.Operation {
	operationID := nextUUID()
	operationName := resourceName + "/operations/" + operationID

	op := postgres.Operation{
		Name: operationName,
		Done: true,
	}
	if response != nil {
		data, _ := json.Marshal(response)
		op.Response = data
	} else {
		// For delete operations, provide an empty JSON object as response
		// The SDK requires non-nil Response even for delete operations
		op.Response = []byte("{}")
	}

	s.PostgresOperations[operationName] = op
	return op
}

// createDefaultBranchLocked creates a default branch for a project (caller must hold lock).
func (s *FakeWorkspace) createDefaultBranchLocked(projectName string) {
	now := nowTime()
	branchUID := "br-" + nextUUID()[:20]
	branchName := projectName + "/branches/" + branchUID

	defaultBranch := postgres.Branch{
		Name:       branchName,
		Parent:     projectName,
		Uid:        branchUID,
		CreateTime: now,
		UpdateTime: now,
		Status: &postgres.BranchStatus{
			CurrentState:     postgres.BranchStatusStateReady,
			StateChangeTime:  now,
			Default:          true,
			IsProtected:      false,
			LogicalSizeBytes: 0,
			ForceSendFields:  []string{"Default", "IsProtected", "LogicalSizeBytes"},
		},
	}

	s.PostgresBranches[branchName] = defaultBranch
}
