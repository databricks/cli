package testserver

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"

	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/genieagents"
)

func (s *FakeWorkspace) GenieAgentCreate(req Request) Response {
	defer s.LockUnlock()()

	var body genieagents.GenieAgent
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return genieAgentInvalidRequest(err)
	}

	// The client may pin the id via the genie_agent_id query parameter; the
	// server assigns one otherwise. ALREADY_EXISTS when the id is in use.
	agentId := req.URL.Query().Get("genie_agent_id")
	if agentId == "" {
		agentId, _ = generateGenieSpaceId()
	}
	if _, ok := s.GenieAgents[agentId]; ok || s.GenieAgentIDsInTrash[agentId] {
		return Response{
			StatusCode: 409,
			Body: map[string]string{
				"error_code": "ALREADY_EXISTS",
				"message":    "Genie Agent " + agentId + " already exists",
			},
		}
	}

	// Default to the caller's home folder when parent_path is omitted,
	// matching the documented behavior and GenieSpaceCreate.
	parentPath := body.ParentPath
	if parentPath == "" {
		parentPath = "/Users/" + s.CurrentUser().UserName
	}

	// An explicitly submitted parent_path must exist as a workspace folder; the
	// CLI's default is the bundle's resource folder, which its mkdirs retry
	// creates. The home folder default needs no check.
	if body.ParentPath != "" {
		if _, ok := s.directories[parentPath]; !ok {
			return Response{
				StatusCode: 404,
				Body: map[string]string{
					"error_code": "RESOURCE_DOES_NOT_EXIST",
					"message":    "Parent folder " + parentPath + " does not exist",
				},
			}
		}
	}

	agent := body
	agent.Name = "agents/" + agentId
	// The API echoes parent_path; effective_parent_path may omit /Workspace.
	agent.ParentPath = body.ParentPath
	agent.EffectiveParentPath = strings.TrimPrefix(parentPath, "/Workspace")
	agent.CreateTime = sdktime.New(time.Now())
	agent.UpdateTime = sdktime.New(time.Now())
	agent.Etag = "1"

	s.GenieAgents[agentId] = agent

	return Response{
		Body: agent,
	}
}

func (s *FakeWorkspace) GenieAgentGet(req Request) Response {
	defer s.LockUnlock()()

	agent, resp := s.getGenieAgent(req.Vars["genie_agent_id"])
	if resp != nil {
		return *resp
	}

	return Response{
		Body: agent,
	}
}

func (s *FakeWorkspace) GenieAgentUpdate(req Request) Response {
	defer s.LockUnlock()()

	agent, resp := s.getGenieAgent(req.Vars["genie_agent_id"])
	if resp != nil {
		return *resp
	}

	if etag := req.URL.Query().Get("etag"); etag != "" && etag != agent.Etag {
		return genieAgentEtagMismatch()
	}

	var body genieagents.GenieAgent
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return genieAgentInvalidRequest(err)
	}

	// The etag covers the definition only, so a metadata-only update keeps it.
	if !reflect.DeepEqual(agent.Definition, body.Definition) {
		next, _ := strconv.Atoi(agent.Etag)
		agent.Etag = strconv.Itoa(next + 1)
	}

	// update_mask "*" replaces every updatable field with the body's value and
	// keeps the server-assigned ones. Setting parent_path moves the agent;
	// clearing it does not (the stored folder is untouched, only the field is
	// cleared, matching the update_mask contract).
	agent.DisplayName = body.DisplayName
	agent.Description = body.Description
	agent.Definition = body.Definition
	agent.WarehouseId = body.WarehouseId
	agent.ParentPath = body.ParentPath
	if body.ParentPath != "" {
		agent.EffectiveParentPath = strings.TrimPrefix(body.ParentPath, "/Workspace")
	}
	agent.UpdateTime = sdktime.New(time.Now())

	s.GenieAgents[req.Vars["genie_agent_id"]] = agent

	return Response{
		Body: agent,
	}
}

func (s *FakeWorkspace) GenieAgentDelete(req Request) Response {
	defer s.LockUnlock()()

	agentId := req.Vars["genie_agent_id"]
	agent, ok := s.GenieAgents[agentId]
	if !ok {
		return genieAgentNotFound()
	}
	if etag := req.URL.Query().Get("etag"); etag != "" && etag != agent.Etag {
		return genieAgentEtagMismatch()
	}

	delete(s.GenieAgents, agentId)
	s.GenieAgentIDsInTrash[agentId] = true

	return Response{
		StatusCode: 200,
	}
}

// getGenieAgent returns a copy of the stored agent, or a 404 response.
func (s *FakeWorkspace) getGenieAgent(agentId string) (genieagents.GenieAgent, *Response) {
	agent, ok := s.GenieAgents[agentId]
	if !ok {
		resp := genieAgentNotFound()
		return genieagents.GenieAgent{}, &resp
	}
	return agent, nil
}

func genieAgentNotFound() Response {
	return Response{
		StatusCode: 404,
		Body: map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Genie Agent not found",
		},
	}
}

func genieAgentEtagMismatch() Response {
	return Response{
		StatusCode: 409,
		Body: map[string]string{
			"error_code": "ABORTED",
			"message":    "Genie Agent definition changed; get the agent and retry",
		},
	}
}

func genieAgentInvalidRequest(err error) Response {
	return Response{
		StatusCode: 400,
		Body: map[string]string{
			"error_code": "INVALID_PARAMETER_VALUE",
			"message":    "Invalid request body: " + err.Error(),
		},
	}
}

// GenieAgentMove simulates a browser move, which does not change the echoed parent_path.
func (s *FakeWorkspace) GenieAgentMove(req Request) Response {
	defer s.LockUnlock()()

	agent, resp := s.getGenieAgent(req.Vars["genie_agent_id"])
	if resp != nil {
		return *resp
	}
	var body struct {
		ParentPath string `json:"parent_path"`
	}
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return genieAgentInvalidRequest(err)
	}
	agent.EffectiveParentPath = strings.TrimPrefix(body.ParentPath, "/Workspace")
	s.GenieAgents[req.Vars["genie_agent_id"]] = agent
	return Response{Body: map[string]any{}}
}
