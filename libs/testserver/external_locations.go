package testserver

import (
	"encoding/json"
	"fmt"
	externallocations "github.com/databricks/cli/libs/sdktypes/uc/externallocations/v1"
	"github.com/databricks/cli/libs/utils"
	"net/http"
)

func (s *FakeWorkspace) ExternalLocationsCreate(req Request) Response {
	defer s.LockUnlock()()

	var createRequest externallocations.CreateExternalLocationRequest
	if err := json.Unmarshal(req.Body, &createRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	if utils.Deref(createRequest.Url) == "" {
		return Response{
			Body:       "CreateExternalLocation Missing required field: url",
			StatusCode: http.StatusBadRequest,
		}
	}

	locationInfo := externallocations.ExternalLocationInfo{
		Name:              createRequest.Name,
		Url:               createRequest.Url,
		CredentialName:    createRequest.CredentialName,
		Comment:           createRequest.Comment,
		ReadOnly:          createRequest.ReadOnly,
		EnableFileEvents:  createRequest.EnableFileEvents,
		Fallback:          createRequest.Fallback,
		EncryptionDetails: createRequest.EncryptionDetails,
		FileEventQueue:    createRequest.FileEventQueue,
		CreatedAt:         utils.NonZero(nowMilli()),
		CreatedBy:         s.CurrentUser().UserName,
		UpdatedBy:         s.CurrentUser().UserName,
		MetastoreId:       utils.NonZero(nextUUID()),
		Owner:             s.CurrentUser().UserName,
	}
	locationInfo.UpdatedAt = locationInfo.CreatedAt

	s.ExternalLocations[utils.Deref(createRequest.Name)] = locationInfo
	return Response{
		Body: locationInfo,
	}
}

func (s *FakeWorkspace) ExternalLocationsUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	existing, ok := s.ExternalLocations[name]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("external location %s not found", name),
		}
	}

	fields, errResponse := parseUpdateFields(req.Body)
	if errResponse != nil {
		return *errResponse
	}

	var updateRequest externallocations.UpdateExternalLocationRequest
	if err := json.Unmarshal(req.Body, &updateRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	applyUpdatedFields(&existing, updateRequest, fields)

	// These three are always sent, so they track the request whether or not the payload
	// names them (read_only=false and a dropped key mean the same thing here).
	existing.ReadOnly = updateRequest.ReadOnly
	existing.EnableFileEvents = updateRequest.EnableFileEvents
	existing.Fallback = updateRequest.Fallback

	if utils.Deref(updateRequest.NewName) != "" {
		existing.Name = updateRequest.NewName

		// Delete the old entry and create with new name
		delete(s.ExternalLocations, name)
		name = utils.Deref(updateRequest.NewName)
	}

	existing.UpdatedAt = utils.NonZero(nowMilli())
	existing.UpdatedBy = s.CurrentUser().UserName

	s.ExternalLocations[name] = existing
	return Response{
		Body: existing,
	}
}
