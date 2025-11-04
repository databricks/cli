package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"dario.cat/mergo"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func (s *FakeWorkspace) SchemasCreate(req Request) Response {
	defer s.LockUnlock()()

	var schema catalog.SchemaInfo

	if err := json.Unmarshal(req.Body, &schema); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	schema.FullName = schema.CatalogName + "." + schema.Name
	s.Schemas[schema.FullName] = schema
	return Response{
		Body: schema,
	}
}

func (s *FakeWorkspace) SchemasUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	existing, ok := s.Schemas[name]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	var schemaUpdate catalog.SchemaInfo

	if err := json.Unmarshal(req.Body, &schemaUpdate); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	err := mergo.Merge(&existing, schemaUpdate, mergo.WithOverride)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("mergo error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	s.Schemas[name] = existing
	return Response{
		Body: existing,
	}
}

/*
func (s *FakeWorkspace) SchemasUpdateGrants(req Request, fullName string) Response {
	var request catalog.UpdatePermissions
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}
	defer s.LockUnlock()()

	// For simplicity, we'll just replace all grants (similar to how job permissions work)
	var grants []catalog.PrivilegeAssignment
	for _, change := range request.Changes {
		if len(change.Add) > 0 {
			grants = append(grants, catalog.PrivilegeAssignment{
				Principal:  change.Principal,
				Privileges: change.Add,
			})
		}
	}
	s.SchemasGrants[fullName] = grants

	return Response{
		Body: catalog.GetPermissionsResponse{
			PrivilegeAssignments: grants,
		},
	}
}

func (s *FakeWorkspace) SchemasGetGrants(req Request, fullName string) Response {
	defer s.LockUnlock()()

	grants := s.SchemasGrants[fullName]
	return Response{
		Body: catalog.GetPermissionsResponse{
			PrivilegeAssignments: grants,
		},
	}
}
*/
