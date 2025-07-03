package testserver

import (
	"encoding/json"
	"fmt"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"
)

func (s *FakeWorkspace) DatabaseInstanceCreate(req Request) Response {
	defer s.LockUnlock()()

	databaseInstance := database.DatabaseInstance{}
	err := json.Unmarshal(req.Body, &databaseInstance)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	databaseInstance.Uid = uuid.New().String()
	s.DatabaseInstances[databaseInstance.Name] = databaseInstance

	return Response{
		Body: databaseInstance,
	}
}
