package testserver

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/common/types/duration"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

func nowTime() *sdktime.Time {
	return sdktime.New(time.Now().UTC())
}

// Backend defaults applied to a project's status when the spec omits them.
// These mirror the values the real API materializes on create.
const (
	defaultHistoryRetention      = 604800 * time.Second // 7 days
	defaultSuspendTimeout        = 86400 * time.Second  // 1 day
	defaultAutoscalingLimitMinCu = 1
	defaultAutoscalingLimitMaxCu = 1
)

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

// postgresManagedByParentErrorResponse creates a 400 BAD_REQUEST response that
// mirrors the real backend's "managed by parent" payload — including the
// details[].ErrorInfo.metadata.declarative_context=MANAGED_BY_PARENT marker
// that declarative clients (TF provider, direct engine) use to disregard the
// delete and rely on the parent to cascade.
func postgresManagedByParentErrorResponse(message string) Response {
	return Response{
		StatusCode: 400,
		Body: map[string]any{
			"error_code": "BAD_REQUEST",
			"message":    message,
			"details": []any{
				map[string]any{
					"@type":  "type.googleapis.com/google.rpc.ErrorInfo",
					"domain": "databricks.com",
					"reason": "RESOURCE_CONFLICT",
					"metadata": map[string]string{
						"declarative_context": "MANAGED_BY_PARENT",
					},
				},
				map[string]any{
					"@type":        "type.googleapis.com/google.rpc.RequestInfo",
					"request_id":   nextUUID(),
					"serving_data": "",
				},
			},
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
		// The backend materializes a default retention when the spec omits it.
		historyRetention := project.Spec.HistoryRetentionDuration
		if historyRetention == nil {
			historyRetention = duration.New(defaultHistoryRetention)
		}

		project.Status = &postgres.ProjectStatus{
			ProjectId:                   projectID,
			DefaultBranch:               name + "/branches/production",
			DisplayName:                 project.Spec.DisplayName,
			PgVersion:                   project.Spec.PgVersion,
			HistoryRetentionDuration:    historyRetention,
			EnablePgNativeLogin:         false,
			Owner:                       TestUser.UserName,
			BranchLogicalSizeLimitBytes: 8796093022208, // 8 TB (real API default)
			SyntheticStorageSizeBytes:   0,
			ForceSendFields:             []string{"EnablePgNativeLogin", "SyntheticStorageSizeBytes"},
		}

		// The backend materializes default endpoint settings when the spec omits them.
		project.Status.DefaultEndpointSettings = &postgres.ProjectDefaultEndpointSettings{
			AutoscalingLimitMinCu:  defaultAutoscalingLimitMinCu,
			AutoscalingLimitMaxCu:  defaultAutoscalingLimitMaxCu,
			SuspendTimeoutDuration: duration.New(defaultSuspendTimeout),
		}
		if project.Spec.DefaultEndpointSettings != nil {
			if project.Spec.DefaultEndpointSettings.AutoscalingLimitMinCu != 0 {
				project.Status.DefaultEndpointSettings.AutoscalingLimitMinCu = project.Spec.DefaultEndpointSettings.AutoscalingLimitMinCu
			}
			if project.Spec.DefaultEndpointSettings.AutoscalingLimitMaxCu != 0 {
				project.Status.DefaultEndpointSettings.AutoscalingLimitMaxCu = project.Spec.DefaultEndpointSettings.AutoscalingLimitMaxCu
			}
			if project.Spec.DefaultEndpointSettings.SuspendTimeoutDuration != nil {
				project.Status.DefaultEndpointSettings.SuspendTimeoutDuration = project.Spec.DefaultEndpointSettings.SuspendTimeoutDuration
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

// PostgresProjectDelete deletes a postgres project. The `purge` query parameter
// is ignored: acceptance tests assert on the recorded HTTP request rather than
// on retention semantics, so a single "remove from map" action serves both
// hard- and soft-delete paths.
func (s *FakeWorkspace) PostgresProjectDelete(name string) Response {
	defer s.LockUnlock()()

	if _, exists := s.PostgresProjects[name]; !exists {
		return postgresNotFoundResponse("project")
	}

	// Deleting a project cascade-deletes all its branches and endpoints.
	branchPrefix := name + "/branches/"
	for brName := range s.PostgresBranches {
		if strings.HasPrefix(brName, branchPrefix) {
			delete(s.PostgresBranches, brName)
			delete(s.postgresImplicitBranches, brName)
			endpointPrefix := brName + "/endpoints/"
			for epName := range s.PostgresEndpoints {
				if strings.HasPrefix(epName, endpointPrefix) {
					delete(s.PostgresEndpoints, epName)
					delete(s.postgresImplicitEndpoints, epName)
				}
			}
		}
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

	if branch.Status != nil && branch.Status.IsProtected {
		return postgresErrorResponse(400, "BAD_REQUEST", "cannot delete protected branch")
	}

	// Branches the server provisioned implicitly (the project's root branch)
	// cannot be deleted independently; they are removed when the project is
	// deleted. Matches the real backend's error payload including the
	// declarative_context=MANAGED_BY_PARENT marker.
	if s.postgresImplicitBranches[name] {
		return postgresManagedByParentErrorResponse(fmt.Sprintf("Cannot delete root branch '%s'. Root branches cannot be deleted independently.", name))
	}

	// Deleting a branch cascade-deletes its endpoints (including the implicit
	// primary).
	endpointPrefix := name + "/endpoints/"
	for epName := range s.PostgresEndpoints {
		if strings.HasPrefix(epName, endpointPrefix) {
			delete(s.PostgresEndpoints, epName)
			delete(s.postgresImplicitEndpoints, epName)
		}
	}

	delete(s.PostgresBranches, name)
	delete(s.postgresImplicitBranches, name)

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

	// Endpoints the server provisioned implicitly (the primary read-write
	// endpoint on every branch) cannot be deleted independently; they are
	// removed when the parent branch is deleted. Matches the real backend's
	// error payload including the declarative_context=MANAGED_BY_PARENT marker.
	if s.postgresImplicitEndpoints[name] {
		return postgresManagedByParentErrorResponse(fmt.Sprintf("Cannot delete read-write endpoint '%s'. Read-write endpoints cannot be deleted.", name))
	}

	delete(s.PostgresEndpoints, name)
	delete(s.postgresImplicitEndpoints, name)

	return Response{
		Body: s.createOperationLocked(name, nil),
	}
}

// PostgresCatalogCreate creates a new postgres catalog.
func (s *FakeWorkspace) PostgresCatalogCreate(req Request, catalogID string) Response {
	defer s.LockUnlock()()

	if catalogID == "" {
		return postgresErrorResponse(400, "INVALID_PARAMETER_VALUE", `Field 'catalog_id' is required, expected non-default value (not "")!`)
	}

	// The SDK sends request.Catalog (the inner struct) as the body — NOT a
	// {"catalog": ...} wrapper. Unmarshal directly into postgres.Catalog.
	var catalog postgres.Catalog
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &catalog); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			}
		}
	}

	name := "catalogs/" + catalogID

	if _, exists := s.PostgresCatalogs[name]; exists {
		return postgresErrorResponse(409, "ALREADY_EXISTS", "catalog with such id already exists")
	}

	now := nowTime()
	catalog.Name = name
	catalog.Uid = nextUUID()
	catalog.CreateTime = now
	catalog.UpdateTime = now

	status := &postgres.CatalogCatalogStatus{}
	if catalog.Spec != nil {
		status.Branch = catalog.Spec.Branch
		status.PostgresDatabase = catalog.Spec.PostgresDatabase
		// Project portion of the status is "projects/{project_id}", derived
		// from the branch name "projects/{project_id}/branches/{branch_id}".
		if idx := strings.Index(catalog.Spec.Branch, "/branches/"); idx > 0 {
			status.Project = catalog.Spec.Branch[:idx]
		}
	}
	catalog.Status = status

	// Real API only returns status on GET (no spec). Match that to keep
	// acceptance test output stable between local and cloud.
	catalog.Spec = nil

	s.PostgresCatalogs[name] = catalog

	return Response{Body: s.createOperationLocked(name, catalog)}
}

// PostgresCatalogGet retrieves a postgres catalog by name.
func (s *FakeWorkspace) PostgresCatalogGet(name string) Response {
	defer s.LockUnlock()()

	catalog, exists := s.PostgresCatalogs[name]
	if !exists {
		return postgresNotFoundResponse("catalog")
	}
	return Response{Body: catalog}
}

// PostgresCatalogDelete deletes a postgres catalog.
func (s *FakeWorkspace) PostgresCatalogDelete(name string) Response {
	defer s.LockUnlock()()

	if _, exists := s.PostgresCatalogs[name]; !exists {
		return postgresNotFoundResponse("catalog")
	}
	delete(s.PostgresCatalogs, name)
	return Response{Body: s.createOperationLocked(name, nil)}
}

// roleIDPattern matches a valid postgres role_id per RFC 1123: lowercase letters,
// numbers and hyphens; 4-63 chars; must start with a letter.
var roleIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{3,62}$`)

// roleStatusFromSpec mirrors the real Postgres Role server's behavior of echoing
// the spec onto Status (plus default-deriving fields the user did not specify)
// while leaving Spec=nil on GET responses.
func roleStatusFromSpec(spec *postgres.RoleRoleSpec) *postgres.RoleRoleStatus {
	status := &postgres.RoleRoleStatus{}
	if spec == nil {
		return status
	}
	status.PostgresRole = spec.PostgresRole
	status.MembershipRoles = spec.MembershipRoles
	status.IdentityType = spec.IdentityType
	if status.IdentityType == "" {
		// Server returns IDENTITY_TYPE_UNSPECIFIED for plain Postgres roles.
		status.IdentityType = "IDENTITY_TYPE_UNSPECIFIED"
	}
	status.AuthMethod = spec.AuthMethod
	if status.AuthMethod == "" {
		// Server derives auth_method from identity_type when the user omits it:
		// see SDK comment on postgres.RoleRoleSpec.AuthMethod.
		switch spec.IdentityType {
		case postgres.RoleIdentityTypeGroup:
			status.AuthMethod = postgres.RoleAuthMethodNoLogin
		case postgres.RoleIdentityTypeUser, postgres.RoleIdentityTypeServicePrincipal:
			status.AuthMethod = postgres.RoleAuthMethodLakebaseOauthV1
		default:
			status.AuthMethod = postgres.RoleAuthMethodPgPasswordScramSha256
		}
	}
	// Real server always echoes an attributes block (all-false when unspecified).
	attrs := &postgres.RoleAttributes{
		ForceSendFields: []string{"Bypassrls", "Createdb", "Createrole"},
	}
	if spec.Attributes != nil {
		attrs.Bypassrls = spec.Attributes.Bypassrls
		attrs.Createdb = spec.Attributes.Createdb
		attrs.Createrole = spec.Attributes.Createrole
	}
	status.Attributes = attrs
	return status
}

// PostgresRoleCreate creates a new postgres role.
func (s *FakeWorkspace) PostgresRoleCreate(req Request, parent, roleID string) Response {
	defer s.LockUnlock()()

	// Check if parent branch exists
	if _, exists := s.PostgresBranches[parent]; !exists {
		return postgresNotFoundResponse("branch")
	}

	// When role_id is empty the real API generates one; mirror that here so the
	// CLI's "let the server pick" path is exercised by tests.
	if roleID == "" {
		roleID = "role-" + nextUUID()[:8]
	}
	if !roleIDPattern.MatchString(roleID) {
		return postgresErrorResponse(400, "INVALID_PARAMETER_VALUE",
			`Field 'role_id' must be 4-63 characters, start with a lowercase letter, and contain only lowercase letters, numbers and hyphens (RFC 1123).`)
	}

	var role postgres.Role
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &role); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			}
		}
	}

	name := fmt.Sprintf("%s/roles/%s", parent, roleID)

	if _, exists := s.PostgresRoles[name]; exists {
		return postgresErrorResponse(409, "ALREADY_EXISTS", "role with such id already exists")
	}

	now := nowTime()
	role.Name = name
	role.Parent = parent
	role.CreateTime = now
	role.UpdateTime = now

	role.Status = roleStatusFromSpec(role.Spec)
	role.Status.RoleId = roleID
	role.Spec = nil

	s.PostgresRoles[name] = role

	return Response{
		Body: s.createOperationLocked(role.Name, role),
	}
}

// PostgresRoleGet retrieves a postgres role by name.
func (s *FakeWorkspace) PostgresRoleGet(name string) Response {
	defer s.LockUnlock()()

	// Extract project and branch names from role name
	// Format: projects/{project}/branches/{branch}/roles/{role}
	parts := strings.Split(name, "/branches/")
	if len(parts) == 2 {
		projectName := parts[0]
		if _, exists := s.PostgresProjects[projectName]; !exists {
			return postgresNotFoundResponse("project")
		}
		branchParts := strings.Split(parts[1], "/roles/")
		if len(branchParts) == 2 {
			branchName := projectName + "/branches/" + branchParts[0]
			if _, exists := s.PostgresBranches[branchName]; !exists {
				return postgresNotFoundResponse("branch")
			}
		}
	}

	role, exists := s.PostgresRoles[name]
	if !exists {
		return postgresNotFoundResponse("role")
	}

	return Response{
		Body: role,
	}
}

// PostgresRoleList lists all postgres roles for a branch.
func (s *FakeWorkspace) PostgresRoleList(parent string) Response {
	defer s.LockUnlock()()

	if _, exists := s.PostgresBranches[parent]; !exists {
		return postgresNotFoundResponse("branch")
	}

	var roles []postgres.Role
	prefix := parent + "/roles/"
	for name, r := range s.PostgresRoles {
		if strings.HasPrefix(name, prefix) {
			roles = append(roles, r)
		}
	}

	return Response{
		Body: postgres.ListRolesResponse{
			Roles: roles,
		},
	}
}

// PostgresRoleUpdate updates a postgres role. The update_mask query parameter
// selects which spec fields to apply; the request body always carries the full
// desired spec. An empty update_mask updates all fields, matching the API
// ("if unspecified, all fields will be updated when possible").
func (s *FakeWorkspace) PostgresRoleUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	role, exists := s.PostgresRoles[name]
	if !exists {
		return postgresNotFoundResponse("role")
	}

	var updateRole postgres.Role
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &updateRole); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			}
		}
	}

	if updateRole.Spec != nil {
		// Preserve role_id which is derived from the resource name.
		roleID := ""
		if role.Status != nil {
			roleID = role.Status.RoleId
		}
		var paths []string
		if mask := req.URL.Query().Get("update_mask"); mask != "" {
			paths = strings.Split(mask, ",")
		}
		role.Status = applyRoleSpecMask(role.Status, roleStatusFromSpec(updateRole.Spec), paths)
		role.Status.RoleId = roleID
	}

	role.UpdateTime = nowTime()
	s.PostgresRoles[name] = role

	return Response{
		Body: s.createOperationLocked(role.Name, role),
	}
}

// applyRoleSpecMask returns the role status after applying the fields named in
// paths (the update_mask) from desired onto existing. Paths are relative to the
// Role and "spec."-prefixed (e.g. "spec.attributes.createdb"); the bare path
// "spec" updates the whole subtree. An empty paths slice updates everything.
//
// Because the request body always carries the full desired spec, a nested path is
// reduced to its top-level spec field — applying the whole field is equivalent.
func applyRoleSpecMask(existing, desired *postgres.RoleRoleStatus, paths []string) *postgres.RoleRoleStatus {
	if len(paths) == 0 || existing == nil {
		return desired
	}
	result := *existing
	for _, p := range paths {
		field, _, _ := strings.Cut(strings.TrimPrefix(p, "spec."), ".")
		switch field {
		case "spec":
			result = *desired
		case "postgres_role":
			result.PostgresRole = desired.PostgresRole
		case "auth_method":
			result.AuthMethod = desired.AuthMethod
		case "identity_type":
			result.IdentityType = desired.IdentityType
		case "membership_roles":
			result.MembershipRoles = desired.MembershipRoles
		case "attributes":
			result.Attributes = desired.Attributes
		}
	}
	return &result
}

// PostgresRoleDelete deletes a postgres role.
func (s *FakeWorkspace) PostgresRoleDelete(name string) Response {
	defer s.LockUnlock()()

	if _, exists := s.PostgresRoles[name]; !exists {
		return postgresNotFoundResponse("role")
	}

	delete(s.PostgresRoles, name)

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

	// Determine resource type from name for metadata @type.
	// Check the more specific suffixes first since role/endpoint names also
	// contain "/branches/".
	resourceType := "Project"
	switch {
	case strings.HasPrefix(resourceName, "catalogs/"):
		resourceType = "Catalog"
	case strings.HasPrefix(resourceName, "synced_tables/"):
		resourceType = "SyncedTable"
	case strings.Contains(resourceName, "/endpoints/"):
		resourceType = "Endpoint"
	case strings.Contains(resourceName, "/roles/"):
		resourceType = "Role"
	case strings.Contains(resourceName, "/branches/"):
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

// PostgresSyncedTableCreate creates a new postgres synced table.
func (s *FakeWorkspace) PostgresSyncedTableCreate(req Request, syncedTableID string) Response {
	defer s.LockUnlock()()

	if syncedTableID == "" {
		return postgresErrorResponse(400, "INVALID_PARAMETER_VALUE", `Field 'synced_table_id' is required, expected non-default value (not "")!`)
	}

	var table postgres.SyncedTable
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &table); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("cannot unmarshal request body: %v", err),
			}
		}
	}

	name := "synced_tables/" + syncedTableID

	if _, exists := s.PostgresSyncedTables[name]; exists {
		return postgresErrorResponse(409, "ALREADY_EXISTS", "synced table with such id already exists")
	}
	table.Name = name
	table.Uid = nextUUID()
	table.CreateTime = nowTime()

	// GET on the real API returns only status; clear spec to match.
	table.Spec = nil
	table.Status = &postgres.SyncedTableSyncedTableStatus{
		DetailedState:                 postgres.SyncedTableStateSyncedTableOnline,
		UnityCatalogProvisioningState: postgres.ProvisioningInfoStateActive,
	}

	s.PostgresSyncedTables[name] = table

	return Response{Body: s.createOperationLocked(name, table)}
}

// PostgresSyncedTableGet retrieves a postgres synced table by name.
func (s *FakeWorkspace) PostgresSyncedTableGet(name string) Response {
	defer s.LockUnlock()()

	table, exists := s.PostgresSyncedTables[name]
	if !exists {
		return postgresNotFoundResponse("synced table")
	}
	return Response{Body: table}
}

// PostgresSyncedTableDelete deletes a postgres synced table.
func (s *FakeWorkspace) PostgresSyncedTableDelete(name string) Response {
	defer s.LockUnlock()()

	if _, exists := s.PostgresSyncedTables[name]; !exists {
		return postgresNotFoundResponse("synced table")
	}
	delete(s.PostgresSyncedTables, name)
	return Response{Body: s.createOperationLocked(name, nil)}
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
	s.postgresImplicitBranches[branchName] = true

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
	s.postgresImplicitEndpoints[endpointName] = true
}
