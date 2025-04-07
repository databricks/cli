package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"dario.cat/mergo"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func (s *FakeWorkspace) SchemasCreate(req Request) Response {
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

	fmt.Fprintf(os.Stderr, "Merging %#v\ninto\n%#v\n\n", schemaUpdate, existing)
	// AI TODO: explain why this Merge operation did not copy Comment from schemaUpdate into existing
	err := mergo.Merge(&existing, schemaUpdate)
	fmt.Fprintf(os.Stderr, "MERGED %#v\ninto\n%#v\n\n", schemaUpdate, existing)
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
