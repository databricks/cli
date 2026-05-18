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

// postgresErrorResponse creates an error response with error code for postgres API.
func postgresErrorResponse(statusCode int, errorCode, message string) Response {
	return Response{
		StatusCode: statusCode,
		Body: map[string]string{
			"error_code": errorCode,
			"message":    message,
		},
	}
}

// postgresNotFoundResponse creates a NOT_FOUND error response for a resource type.
func postgresNotFoundResponse(resourceType string) Response {
	// Include trace ID to match real API behavior for not found errors
	message := resourceType + " id not found [TraceId: " + nextUUID() + "]"
	return Response{
		StatusCode: 404,
		Body: map[string]string{
			"error_code": "NOT_FOUND",
			"message":    message,
		},
	}
}

// PostgresProjectCreate creates a new postgres project.
func (s *FakeWorkspace) PostgresProjectCreate(req Request, projectID string) Response {
	defer s.LockUnlock()()

	// Validate that project_id is provided
	if projectID == "" {
		return postgresErrorResponse(400, "INVALID_PARAMETER_VALUE", `Field 'project_id' is required, expected non-default value (not "")!`)
	}

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
		return postgresErrorResponse(409, "ALREADY_EXISTS", "project with such id already exists in the workspace")
	}

	now := nowTime()
	project.Name = name
	project.Uid = nextUUID()
	project.CreateTime = now
	project.UpdateTime = now

	// Copy spec fields to status (API returns status as materialized view)
	if project.Spec != nil {
		project.Status = &postgres.ProjectStatus{
			ProjectId:                   projectID,
			DefaultBranch:               name + "/branches/production",
			DisplayName:                 project.Spec.DisplayName,
			PgVersion:                   project.Spec.PgVersion,
			HistoryRetentionDuration:    project.Spec.HistoryRetentionDuration,
			EnablePgNativeLogin:         true,
			Owner:                       TestUser.UserName,
			BranchLogicalSizeLimitBytes: 8796093022208, // 8 TB (real API default)
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
		return postgresNotFoundResponse("project")
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
		return postgresNotFoundResponse("project")
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
		return postgresNotFoundResponse("project")
	}

	delete(s.PostgresProjects, name)

	return Response{
		Body: s.createOperationLocked(name, nil),
	}
}

// PostgresBranchCreate creates a new postgres branch.
//
// When replaceExisting is true, an existing branch with the same ID is updated
// in place instead of returning ALREADY_EXISTS. This mirrors the backend behavior
// that lets users bring the implicitly-created production branch under management.
func (s *FakeWorkspace) PostgresBranchCreate(req Request, parent, branchID string, replaceExisting bool) Response {
	defer s.LockUnlock()()

	// Validate that branch_id is provided
	if branchID == "" {
		return postgresErrorResponse(400, "INVALID_PARAMETER_VALUE", `Field 'branch_id' is required, expected non-default value (not "")!`)
	}

	// Check if parent project exists
	if _, exists := s.PostgresProjects[parent]; !exists {
		return postgresNotFoundResponse("project")
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

	existing, exists := s.PostgresBranches[name]
	if exists && !replaceExisting {
		return postgresErrorResponse(409, "ALREADY_EXISTS", "branch with such id already exists")
	}

	now := nowTime()
	if exists {
		// Preserve identifying / output-only fields; apply incoming spec to status.
		branch.Name = existing.Name
		branch.Parent = existing.Parent
		branch.Uid = existing.Uid
		branch.CreateTime = existing.CreateTime
		branch.UpdateTime = now
		branch.Status = existing.Status
	} else {
		branch.Name = name
		branch.Parent = parent
		branch.Uid = "br-" + nextUUID()[:20]
		branch.CreateTime = now
		branch.UpdateTime = now
		branch.Status = &postgres.BranchStatus{
			BranchId:         branchID,
			CurrentState:     postgres.BranchStatusStateReady,
			StateChangeTime:  now,
			Default:          false,
			IsProtected:      false,
			LogicalSizeBytes: 0,
			ForceSendFields:  []string{"Default", "IsProtected", "LogicalSizeBytes"},
		}

		// Set source branch info if a default branch exists.
		prefix := parent + "/branches/"
		for brName, br := range s.PostgresBranches {
			if strings.HasPrefix(brName, prefix) && br.Status != nil && br.Status.Default {
				branch.Status.SourceBranch = br.Name
				branch.Status.SourceBranchLsn = "0/0"
				branch.Status.SourceBranchTime = nowTime()
				break
			}
		}
	}

	// Apply user-provided spec fields to status (where input fields are surfaced).
	if branch.Spec != nil {
		branch.Status.IsProtected = branch.Spec.IsProtected
	}

	// Clear spec - API only returns status
	branch.Spec = nil

	s.PostgresBranches[name] = branch

	// Each branch implicitly provisions a primary read-write endpoint. Only create
	// it on first branch creation; on replace_existing the primary already exists.
	if !exists {
		s.createDefaultEndpointLocked(name)
	}

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
			return postgresNotFoundResponse("project")
		}
	}

	branch, exists := s.PostgresBranches[name]
	if !exists {
		return postgresNotFoundResponse("branch")
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
		return postgresNotFoundResponse("project")
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
		return postgresNotFoundResponse("branch")
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
		return postgresNotFoundResponse("branch")
	}

	// Check if branch is protected
	if branch.Status != nil && branch.Status.IsProtected {
		return postgresErrorResponse(400, "BAD_REQUEST", "cannot delete protected branch")
	}

	delete(s.PostgresBranches, name)

	return Response{
		Body: s.createOperationLocked(name, nil),
	}
}

// PostgresEndpointCreate creates a new postgres endpoint.
//
// When replaceExisting is true, an existing endpoint with the same ID is updated
// in place instead of returning ALREADY_EXISTS. This mirrors the backend behavior
// that lets users bring the implicitly-created primary read-write endpoint under
// management.
func (s *FakeWorkspace) PostgresEndpointCreate(req Request, parent, endpointID string, replaceExisting bool) Response {
	defer s.LockUnlock()()

	// Validate that endpoint_id is provided
	if endpointID == "" {
		return postgresErrorResponse(400, "INVALID_PARAMETER_VALUE", `Field 'endpoint_id' is required, expected non-default value (not "")!`)
	}

	// Check if parent branch exists
	branch, exists := s.PostgresBranches[parent]
	if !exists {
		return postgresNotFoundResponse("branch")
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

	existing, exists := s.PostgresEndpoints[name]
	if exists && !replaceExisting {
		return postgresErrorResponse(409, "ALREADY_EXISTS", "endpoint with such id already exists")
	}

	now := nowTime()
	if exists {
		endpoint.Name = existing.Name
		endpoint.Parent = existing.Parent
		endpoint.Uid = existing.Uid
		endpoint.CreateTime = existing.CreateTime
		endpoint.UpdateTime = now
		endpoint.Status = existing.Status
	} else {
		endpoint.Name = name
		endpoint.Parent = parent
		endpoint.Uid = "ep-" + nextUUID()[:20]
		endpoint.CreateTime = now
		endpoint.UpdateTime = now

		// Inherit suspend timeout default from the project.
		var defaultSuspendTimeout *duration.Duration
		if project, ok := s.PostgresProjects[branch.Parent]; ok {
			if project.Status != nil && project.Status.DefaultEndpointSettings != nil {
				defaultSuspendTimeout = project.Status.DefaultEndpointSettings.SuspendTimeoutDuration
			}
		}

		endpoint.Status = &postgres.EndpointStatus{
			EndpointId:             endpointID,
			CurrentState:           postgres.EndpointStatusStateActive,
			Disabled:               false,
			Settings:               &postgres.EndpointSettings{},
			SuspendTimeoutDuration: defaultSuspendTimeout,
			ForceSendFields:        []string{"Disabled"},
		}

		host := endpoint.Uid + ".database.us-east-1.cloud.databricks.com"
		endpoint.Status.Hosts = &postgres.EndpointHosts{
			Host:         host,
			ReadOnlyHost: host,
		}
		endpoint.Status.Group = &postgres.EndpointGroupStatus{
			EnableReadableSecondaries: true,
			Max:                       1,
			Min:                       1,
		}
	}

	// Apply user-provided spec to status (where input fields are surfaced).
	if endpoint.Spec != nil {
		endpoint.Status.EndpointType = endpoint.Spec.EndpointType
		endpoint.Status.AutoscalingLimitMinCu = endpoint.Spec.AutoscalingLimitMinCu
		endpoint.Status.AutoscalingLimitMaxCu = endpoint.Spec.AutoscalingLimitMaxCu
		if endpoint.Spec.SuspendTimeoutDuration != nil {
			endpoint.Status.SuspendTimeoutDuration = endpoint.Spec.SuspendTimeoutDuration
		}
		endpoint.Status.Disabled = endpoint.Spec.Disabled
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
			return postgresNotFoundResponse("project")
		}
		// Check if branch exists
		branchParts := strings.Split(parts[1], "/endpoints/")
		if len(branchParts) == 2 {
			branchName := projectName + "/branches/" + branchParts[0]
			if _, exists := s.PostgresBranches[branchName]; !exists {
				return postgresNotFoundResponse("branch")
			}
		}
	}

	endpoint, exists := s.PostgresEndpoints[name]
	if !exists {
		return postgresNotFoundResponse("endpoint")
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
		return postgresNotFoundResponse("branch")
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
		return postgresNotFoundResponse("endpoint")
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
		if updateEndpoint.Spec.SuspendTimeoutDuration != nil {
			endpoint.Status.SuspendTimeoutDuration = updateEndpoint.Spec.SuspendTimeoutDuration
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
		return postgresNotFoundResponse("endpoint")
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
		return postgresNotFoundResponse("operation")
	}

	return Response{
		Body: operation,
	}
}

// createOperationLocked creates and stores an operation (caller must hold lock).
func (s *FakeWorkspace) createOperationLocked(resourceName string, response any) postgres.Operation {
	operationID := nextUUID()
	operationName := resourceName + "/operations/" + operationID

	// Determine resource type from name for metadata @type
	resourceType := "Project"
	if strings.Contains(resourceName, "/endpoints/") {
		resourceType = "Endpoint"
	} else if strings.Contains(resourceName, "/branches/") {
		resourceType = "Branch"
	}

	op := postgres.Operation{
		Name:     operationName,
		Done:     true,
		Metadata: fmt.Appendf(nil, `{"@type":"type.googleapis.com/databricks.postgres.v1.%sOperationMetadata"}`, resourceType),
	}
	if response != nil {
		data, _ := json.Marshal(response)
		op.Response = data
	} else {
		// For delete operations, provide Empty response with @type
		op.Response = []byte(`{"@type":"type.googleapis.com/google.protobuf.Empty"}`)
	}

	s.PostgresOperations[operationName] = op
	return op
}

// createDefaultBranchLocked creates a default branch for a project (caller must hold lock).
// The default branch is named "production" to match cloud API behavior.
func (s *FakeWorkspace) createDefaultBranchLocked(projectName string) {
	now := nowTime()
	branchUID := "br-" + nextUUID()[:20]
	branchName := projectName + "/branches/production"

	defaultBranch := postgres.Branch{
		Name:       branchName,
		Parent:     projectName,
		Uid:        branchUID,
		CreateTime: now,
		UpdateTime: now,
		Status: &postgres.BranchStatus{
			BranchId:         "production",
			CurrentState:     postgres.BranchStatusStateReady,
			StateChangeTime:  now,
			Default:          true,
			IsProtected:      false,
			LogicalSizeBytes: 0,
			ForceSendFields:  []string{"Default", "IsProtected", "LogicalSizeBytes"},
		},
	}

	s.PostgresBranches[branchName] = defaultBranch

	// Each branch implicitly provisions a primary read-write endpoint.
	s.createDefaultEndpointLocked(branchName)
}

// createDefaultEndpointLocked creates the implicit primary read-write endpoint for
// a branch (caller must hold lock). The endpoint is named "primary" to match cloud
// API behavior.
func (s *FakeWorkspace) createDefaultEndpointLocked(branchName string) {
	now := nowTime()
	endpointName := branchName + "/endpoints/primary"

	// Inherit default endpoint settings from the project where available.
	var (
		minCu, maxCu   float64
		suspendTimeout *duration.Duration
		projectName    = strings.Split(branchName, "/branches/")[0]
	)
	if project, ok := s.PostgresProjects[projectName]; ok {
		if project.Status != nil && project.Status.DefaultEndpointSettings != nil {
			minCu = project.Status.DefaultEndpointSettings.AutoscalingLimitMinCu
			maxCu = project.Status.DefaultEndpointSettings.AutoscalingLimitMaxCu
			suspendTimeout = project.Status.DefaultEndpointSettings.SuspendTimeoutDuration
		}
	}

	endpointUID := "ep-" + nextUUID()[:20]
	host := endpointUID + ".database.us-east-1.cloud.databricks.com"

	s.PostgresEndpoints[endpointName] = postgres.Endpoint{
		Name:       endpointName,
		Parent:     branchName,
		Uid:        endpointUID,
		CreateTime: now,
		UpdateTime: now,
		Status: &postgres.EndpointStatus{
			EndpointId:             "primary",
			EndpointType:           postgres.EndpointTypeEndpointTypeReadWrite,
			CurrentState:           postgres.EndpointStatusStateActive,
			Disabled:               false,
			Settings:               &postgres.EndpointSettings{},
			AutoscalingLimitMinCu:  minCu,
			AutoscalingLimitMaxCu:  maxCu,
			SuspendTimeoutDuration: suspendTimeout,
			// Primary read-write endpoint exposes only the read-write host;
			// read_only_host is set on read-only endpoints created later.
			Hosts: &postgres.EndpointHosts{Host: host},
			Group: &postgres.EndpointGroupStatus{
				Max:             1,
				Min:             1,
				ForceSendFields: []string{"EnableReadableSecondaries"},
			},
			ForceSendFields: []string{"Disabled"},
		},
	}
}
