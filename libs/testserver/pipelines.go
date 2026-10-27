package testserver

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
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
	if strings.Contains(spec.Target, ".") {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    fmt.Sprintf("CreatePipeline target_schema_name %q is not a valid name. Valid names must contain only alphanumeric characters and underscores, and cannot contain spaces, periods, forward slashes, or control characters.", spec.Target),
			},
		}
	}

	var r pipelines.GetPipelineResponse
	r.Spec = &spec

	// parameters is not on PipelineSpec (only on CreatePipeline), so the decode above
	// drops it. The backend echoes it on GetPipelineResponse.Parameters; mirror that
	// here, else a re-read misses it and the CLI plans a perpetual update.
	var create pipelines.CreatePipeline
	if err := json.Unmarshal(req.Body, &create); err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}
	r.Parameters = create.Parameters

	pipelineId := nextUUID()
	r.PipelineId = pipelineId
	r.CreatorUserName = "tester@databricks.com"
	r.LastModified = nowMilli()
	r.Name = r.Spec.Name
	// run_as is on CreatePipeline, not PipelineSpec, so the spec decode drops it. The backend
	// echoes it top-level on GetPipelineResponse.RunAs; mirror that so a re-read is faithful.
	if create.RunAs != nil {
		r.RunAs = create.RunAs
	}
	r.RunAsUserName = "tester@databricks.com"
	r.State = "IDLE"
	r.EffectivePublishingMode = pipelines.PublishingModeDefaultPublishingMode

	setSpecDefaults(&spec, pipelineId)
	s.Pipelines[pipelineId] = r
	s.setPipelineTestSources(pipelineId, req.Body)

	return Response{
		Body: pipelines.CreatePipelineResponse{
			PipelineId: pipelineId,
		},
	}
}

func setSpecDefaults(spec *pipelines.PipelineSpec, pipelineId string) {
	spec.Id = pipelineId
	// If the pipeline definition does not specify a catalog, it switches to Hive metastore mode
	// and if the storage location is not specified, API automatically generates a storage location
	// (ref: https://docs.databricks.com/gcp/en/dlt/hive-metastore#specify-a-storage-location)
	if spec.Storage == "" && spec.Catalog == "" {
		spec.Storage = "dbfs:/pipelines/" + pipelineId
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
	var edit pipelines.EditPipeline
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
	s.setPipelineTestSources(pipelineId, req.Body)

	return Response{}
}

func (s *FakeWorkspace) PipelineStartUpdate(pipelineId string, body []byte) Response {
	defer s.LockUnlock()()

	pipeline, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}

	for id, update := range s.PipelineUpdates {
		if update.pipelineID == pipelineId && update.active() {
			return Response{
				StatusCode: 409,
				Body: map[string]string{
					"error_code": "INVALID_STATE_TRANSITION",
					"message":    fmt.Sprintf("An active update '%s' already exists for pipeline '%s'.", id, pipelineId),
				},
			}
		}
	}

	var test pipelineTestRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &test); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			}
		}
	}
	if test.TestOptions != nil && !test.TestOnly {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": "test_options requires test_only to be true."},
		}
	}

	updateId := nextUUID()
	update := &fakePipelineUpdate{
		pipelineID: pipelineId,
		states:     []pipelines.UpdateInfoState{pipelines.UpdateInfoStateCompleted},
	}
	if test.TestOnly {
		var selection []string
		if test.TestOptions != nil {
			selection = test.TestOptions.TestSelection
		}
		targets, message := pipelineTestTargets(s.pipelineTestSources[pipelineId], selection)
		if message != "" {
			return Response{
				StatusCode: 400,
				Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": message},
			}
		}
		var rootPath string
		if pipeline.Spec != nil {
			rootPath = pipeline.Spec.RootPath
		}
		rootDir := pipelineTestRootDir(rootPath, s.pipelineTestSources[pipelineId])
		update = s.startPipelineTestRun(pipelineId, updateId, s.collectPipelineTests(rootDir, targets))
	}
	s.PipelineUpdates[updateId] = update

	return Response{
		Body: pipelines.StartUpdateResponse{
			UpdateId: updateId,
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

	update, updateExists := s.PipelineUpdates[updateId]
	if !updateExists || update.pipelineID != pipelineId {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified update %s was not found.", updateId)},
		}
	}

	update.polls++
	return Response{
		Body: pipelines.GetUpdateResponse{
			Update: &pipelines.UpdateInfo{
				UpdateId: updateId,
				State:    update.states[update.stateIndex()],
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

	// Stop is pipeline-wide: it cancels whichever update is active.
	for _, update := range s.PipelineUpdates {
		if update.pipelineID == pipelineId && update.active() {
			update.cancel()
		}
	}

	return Response{
		Body: pipelines.GetPipelineResponse{
			PipelineId: pipelineId,
			State:      pipelines.PipelineStateIdle,
		},
	}
}
