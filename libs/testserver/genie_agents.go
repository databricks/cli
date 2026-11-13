package testserver

import (
	"encoding/json"
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
	if _, ok := s.GenieAgents[agentId]; ok {
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
	agent.ParentPath = parentPath
	agent.EffectiveParentPath = parentPath
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

	var body genieagents.GenieAgent
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return genieAgentInvalidRequest(err)
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
		agent.EffectiveParentPath = body.ParentPath
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
	if _, ok := s.GenieAgents[agentId]; !ok {
		return genieAgentNotFound()
	}

	delete(s.GenieAgents, agentId)

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

func genieAgentInvalidRequest(err error) Response {
	return Response{
		StatusCode: 400,
		Body: map[string]string{
			"error_code": "INVALID_PARAMETER_VALUE",
			"message":    "Invalid request body: " + err.Error(),
		},
	}
}
