package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func (s *FakeWorkspace) ExternalLocationsCreate(req Request) Response {
	defer s.LockUnlock()()

	var createRequest catalog.CreateExternalLocation
	if err := json.Unmarshal(req.Body, &createRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	if createRequest.Url == "" {
		return Response{
			Body:       "CreateExternalLocation Missing required field: url",
			StatusCode: http.StatusBadRequest,
		}
	}

	locationInfo := catalog.ExternalLocationInfo{
		Name:              createRequest.Name,
		Url:               createRequest.Url,
		CredentialName:    createRequest.CredentialName,
		Comment:           createRequest.Comment,
		ReadOnly:          createRequest.ReadOnly,
		EnableFileEvents:  createRequest.EnableFileEvents,
		Fallback:          createRequest.Fallback,
		EncryptionDetails: createRequest.EncryptionDetails,
		FileEventQueue:    createRequest.FileEventQueue,
		CreatedAt:         time.Now().UnixMilli(),
		CreatedBy:         s.CurrentUser().UserName,
		UpdatedAt:         time.Now().UnixMilli(),
		UpdatedBy:         s.CurrentUser().UserName,
		MetastoreId:       nextUUID(),
		Owner:             s.CurrentUser().UserName,
	}

	s.ExternalLocations[createRequest.Name] = locationInfo
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

	var updateRequest catalog.UpdateExternalLocation
	if err := json.Unmarshal(req.Body, &updateRequest); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Update only the fields that can be updated
	if updateRequest.Comment != "" {
		existing.Comment = updateRequest.Comment
	}
	if updateRequest.Url != "" {
		existing.Url = updateRequest.Url
	}
	if updateRequest.CredentialName != "" {
		existing.CredentialName = updateRequest.CredentialName
	}
	if updateRequest.Owner != "" {
		existing.Owner = updateRequest.Owner
	}
	existing.ReadOnly = updateRequest.ReadOnly
	existing.EnableFileEvents = updateRequest.EnableFileEvents
	existing.Fallback = updateRequest.Fallback

	if updateRequest.NewName != "" {
		existing.Name = updateRequest.NewName

		// Delete the old entry and create with new name
		delete(s.ExternalLocations, name)
		name = updateRequest.NewName
	}

	existing.UpdatedAt = time.Now().UnixMilli()
	existing.UpdatedBy = s.CurrentUser().UserName

	s.ExternalLocations[name] = existing
	return Response{
		Body: existing,
	}
}

func (s *FakeWorkspace) ExternalLocationsGet(_ Request, name string) Response {
	defer s.LockUnlock()()

	existing, ok := s.ExternalLocations[name]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("external location %s not found", name),
		}
	}

	return Response{
		Body: existing,
	}
}

func (s *FakeWorkspace) ExternalLocationsDelete(_ Request, name string) Response {
	defer s.LockUnlock()()

	if _, ok := s.ExternalLocations[name]; !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       fmt.Sprintf("external location %s not found", name),
		}
	}

	delete(s.ExternalLocations, name)
	return Response{
		StatusCode: http.StatusOK,
	}
}
