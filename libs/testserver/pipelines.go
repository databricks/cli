package testserver

import (
	"encoding/json"
	"fmt"
	"strings"

	pipelines "github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/databricks/cli/libs/utils"
)

func (s *FakeWorkspace) PipelineGet(pipelineId string) Response {
	defer s.LockUnlock()()

	value, ok := s.Pipelines[pipelineId]
	if !ok {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}
	return Response{
		Body: value,
	}
}

func (s *FakeWorkspace) PipelineCreate(req Request) Response {
	defer s.LockUnlock()()

	spec := pipelines.PipelineSpec{}
	err := json.Unmarshal(req.Body, &spec)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	// Unity Catalog requires target_schema_name to be a single schema segment, so a
	// dotted catalog.schema value is rejected; the catalog belongs in the separate
	// catalog field. Only the dot trips this check, but the backend's canned error
	// also lists dashes and other characters as invalid.
	if strings.Contains(utils.Deref(spec.Target), ".") {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    fmt.Sprintf("CreatePipeline target_schema_name %q is not a valid name. Valid names must contain only alphanumeric characters and underscores, and cannot contain spaces, periods, forward slashes, or control characters.", utils.Deref(spec.Target)),
			},
		}
	}

	var r pipelines.GetPipelineResponse
	r.Spec = &spec

	// parameters is not on PipelineSpec (only on CreatePipeline), so the decode above
	// drops it. The backend echoes it on GetPipelineResponse.Parameters; mirror that
	// here, else a re-read misses it and the CLI plans a perpetual update.
	var create pipelines.CreatePipelineRequest
	if err := json.Unmarshal(req.Body, &create); err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}
	r.Parameters = create.Parameters

	pipelineId := nextUUID()
	r.PipelineId = utils.NonZero(pipelineId)
	r.CreatorUserName = utils.NonZero("tester@databricks.com")
	r.LastModified = utils.NonZero(nowMilli())
	r.Name = r.Spec.Name
	// run_as is on CreatePipeline, not PipelineSpec, so the spec decode drops it. The backend
	// echoes it top-level on GetPipelineResponse.RunAs; mirror that so a re-read is faithful.
	if create.RunAs != nil {
		r.RunAs = create.RunAs
	}
	r.RunAsUserName = utils.NonZero("tester@databricks.com")
	r.State = "IDLE"
	r.EffectivePublishingMode = pipelines.PublishingMode_DefaultPublishingMode

	setSpecDefaults(&spec, pipelineId)
	s.Pipelines[pipelineId] = r

	return Response{
		Body: pipelines.CreatePipelineResponse{
			PipelineId: utils.NonZero(pipelineId),
		},
	}
}

func setSpecDefaults(spec *pipelines.PipelineSpec, pipelineId string) {
	spec.Id = utils.NonZero(pipelineId)
	// If the pipeline definition does not specify a catalog, it switches to Hive metastore mode
	// and if the storage location is not specified, API automatically generates a storage location
	// (ref: https://docs.databricks.com/gcp/en/dlt/hive-metastore#specify-a-storage-location)
	if utils.Deref(spec.Storage) == "" && utils.Deref(spec.Catalog) == "" {
		spec.Storage = utils.NonZero("dbfs:/pipelines/" + pipelineId)
	}
}

func (s *FakeWorkspace) PipelineUpdate(req Request, pipelineId string) Response {
	defer s.LockUnlock()()

	var spec pipelines.PipelineSpec
	err := json.Unmarshal(req.Body, &spec)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 400,
		}
	}

	item, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
		}
	}

	// parameters is on EditPipeline, not PipelineSpec; round-trip it like
	// PipelineCreate does.
	var edit pipelines.EditPipelineRequest
	if err := json.Unmarshal(req.Body, &edit); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 400,
		}
	}

	item.Spec = &spec
	item.Parameters = edit.Parameters
	// The backend echoes the spec name on GetPipelineResponse.Name; mirror that so a
	// rename is reflected on the next read.
	item.Name = spec.Name
	// run_as is on EditPipeline, not PipelineSpec; keep it in sync like Parameters so an edit
	// that changes run_as is reflected on the next read (matches cloud top-level echo).
	if edit.RunAs != nil {
		item.RunAs = edit.RunAs
	}
	setSpecDefaults(&spec, pipelineId)
	s.Pipelines[pipelineId] = item

	return Response{}
}

func (s *FakeWorkspace) PipelineStartUpdate(pipelineId string) Response {
	defer s.LockUnlock()()

	_, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}

	updateId := nextUUID()
	s.PipelineUpdates[updateId] = true

	return Response{
		Body: pipelines.StartUpdateResponse{
			UpdateId: utils.NonZero(updateId),
		},
	}
}

func (s *FakeWorkspace) PipelineEvents(pipelineId string) Response {
	defer s.LockUnlock()()

	_, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}

	return Response{
		Body: map[string]any{
			"events": []pipelines.PipelineEvent{},
		},
	}
}

func (s *FakeWorkspace) PipelineGetUpdate(pipelineId, updateId string) Response {
	defer s.LockUnlock()()

	_, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}

	// Check if the update exists
	_, updateExists := s.PipelineUpdates[updateId]
	if !updateExists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified update %s was not found.", updateId)},
		}
	}

	return Response{
		Body: pipelines.GetUpdateResponse{
			Update: &pipelines.UpdateInfo{
				UpdateId: utils.NonZero(updateId),
				State:    pipelines.UpdateState_Completed,
			},
		},
	}
}

func (s *FakeWorkspace) PipelineStop(pipelineId string) Response {
	defer s.LockUnlock()()

	_, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}

	return Response{
		Body: pipelines.GetPipelineResponse{
			PipelineId: utils.NonZero(pipelineId),
			State:      pipelines.PipelineState_PipelineState_Idle,
		},
	}
}
