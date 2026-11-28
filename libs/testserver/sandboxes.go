package testserver

import (
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"uuid"

	"github.com/databricks/databricks-sdk-go/service/sandbox"
)

var sandboxResourceID = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)
})

func sandboxNotFound() Response {
	return sandboxFilesError(404, "NOT_FOUND", "Sandbox not found")
}

func (s *FakeWorkspace) SandboxesCreate(req Request) Response {
	var sb sandbox.Sandbox
	if err := json.Unmarshal(req.Body, &sb); err != nil {
		return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", "Invalid sandbox")
	}
	id := req.URL.Query().Get("sandbox_id")
	if _, err := uuid.Parse(id); !sandboxResourceID().MatchString(id) || err == nil {
		return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", "Invalid sandbox_id")
	}
	defer s.LockUnlock()()
	if _, exists := s.Sandboxes[id]; exists {
		return sandboxFilesError(409, "RESOURCE_ALREADY_EXISTS", "Sandbox already exists")
	}
	sb.Name = "sandboxes/" + id
	sb.Status = &sandbox.SandboxStatus{State: sandbox.SandboxStateSandboxStatePending}
	s.Sandboxes[id] = sb
	return Response{Body: sb}
}

func (s *FakeWorkspace) SandboxesGet(id string) Response {
	defer s.LockUnlock()()
	sb, ok := s.Sandboxes[id]
	if !ok {
		return sandboxNotFound()
	}
	// Complete accepted provisioning on the first poll, without real-time sleeps.
	if sb.Status != nil && sb.Status.State == sandbox.SandboxStateSandboxStatePending {
		sb.Status = &sandbox.SandboxStatus{State: sandbox.SandboxStateSandboxStateRunning}
		s.Sandboxes[id] = sb
	}
	return Response{Body: sb}
}

func (s *FakeWorkspace) SandboxesList(req Request) Response {
	defer s.LockUnlock()()
	ids := make([]string, 0, len(s.Sandboxes))
	for id := range s.Sandboxes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	query := req.URL.Query()
	pageSize, _ := strconv.Atoi(query.Get("page_size"))
	if pageSize <= 0 || pageSize > 1000 {
		pageSize = 1000
	}
	offset, err := strconv.Atoi(query.Get("page_token"))
	if query.Get("page_token") != "" && (err != nil || offset < 0 || offset > len(ids)) {
		return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", "Invalid page_token")
	}
	end := min(offset+pageSize, len(ids))
	resp := sandbox.ListSandboxesResponse{}
	for _, id := range ids[offset:end] {
		resp.Sandboxes = append(resp.Sandboxes, s.Sandboxes[id])
	}
	if end < len(ids) {
		resp.NextPageToken = strconv.Itoa(end)
	}
	return Response{Body: resp}
}

func (s *FakeWorkspace) SandboxesUpdate(req Request, id string) Response {
	var update sandbox.Sandbox
	if err := json.Unmarshal(req.Body, &update); err != nil {
		return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", "Invalid sandbox")
	}
	mask := req.URL.Query().Get("update_mask")
	if mask == "" {
		return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", "update_mask is required")
	}
	for field := range strings.SplitSeq(mask, ",") {
		if field != "display_name" && field != "spec.compute.inactivity_timeout" {
			return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", "Invalid update_mask")
		}
		if field == "spec.compute.inactivity_timeout" && (update.Spec == nil || update.Spec.Compute == nil || update.Spec.Compute.InactivityTimeout == nil) {
			return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", "inactivity_timeout is required")
		}
	}
	defer s.LockUnlock()()
	sb, ok := s.Sandboxes[id]
	if !ok {
		return sandboxNotFound()
	}
	for field := range strings.SplitSeq(mask, ",") {
		switch field {
		case "display_name":
			sb.DisplayName = update.DisplayName
		case "spec.compute.inactivity_timeout":
			spec := sandbox.SandboxSpec{}
			compute := sandbox.ComputeSpec{}
			if sb.Spec != nil {
				spec = *sb.Spec
				if sb.Spec.Compute != nil {
					compute = *sb.Spec.Compute
				}
			}
			compute.InactivityTimeout = update.Spec.Compute.InactivityTimeout
			spec.Compute = &compute
			sb.Spec = &spec
		}
	}
	s.Sandboxes[id] = sb
	return Response{Body: sb}
}

func (s *FakeWorkspace) SandboxesDelete(id string) Response {
	defer s.LockUnlock()()
	if _, ok := s.Sandboxes[id]; !ok {
		return sandboxNotFound()
	}
	delete(s.Sandboxes, id)
	delete(s.sandboxFiles, id)
	return Response{}
}

func (s *FakeWorkspace) SandboxesSetState(id string, state sandbox.SandboxState) Response {
	defer s.LockUnlock()()
	sb, ok := s.Sandboxes[id]
	if !ok {
		return sandboxNotFound()
	}
	if state == sandbox.SandboxStateSandboxStateRunning && sb.Status != nil && sb.Status.State != sandbox.SandboxStateSandboxStateStopped {
		return sandboxFilesError(400, "FAILED_PRECONDITION", "Sandbox must be stopped before starting")
	}
	if state == sandbox.SandboxStateSandboxStateRunning {
		state = sandbox.SandboxStateSandboxStatePending
	}
	sb.Status = &sandbox.SandboxStatus{State: state}
	s.Sandboxes[id] = sb
	return Response{Body: sb}
}
