package main

import (
	"strings"
	"testing"

	"github.com/databricks/cli/internal/clijson"
)

func TestNormalizeFieldlessResponses(t *testing.T) {
	emptyObject := &clijson.EntityJSON{PascalName: "EmptyObject", IsObject: true}
	tests := []struct {
		name     string
		response *clijson.EntityJSON
		schemas  map[string]*clijson.SchemaJSON
		want     bool
	}{
		{
			name:     "explicit marker is preserved",
			response: &clijson.EntityJSON{PascalName: "FieldfulObject", IsObject: true, IsEmptyResponse: true},
			schemas:  map[string]*clijson.SchemaJSON{"FieldfulObject": {Fields: map[string]*clijson.SchemaFieldJSON{"name": {}}}},
			want:     true,
		},
		{
			name:     "known fieldless object",
			response: emptyObject,
			schemas:  map[string]*clijson.SchemaJSON{"EmptyObject": {Fields: map[string]*clijson.SchemaFieldJSON{}}},
			want:     true,
		},
		{
			name:     "explicit marker on known fieldless object",
			response: &clijson.EntityJSON{PascalName: "EmptyObject", IsObject: true, IsEmptyResponse: true},
			schemas:  map[string]*clijson.SchemaJSON{"EmptyObject": {Fields: map[string]*clijson.SchemaFieldJSON{}}},
			want:     true,
		},
		{
			name:     "unknown schema",
			response: &clijson.EntityJSON{PascalName: "UnknownObject", IsObject: true},
			schemas:  map[string]*clijson.SchemaJSON{},
			want:     false,
		},
		{
			name:     "non-object response",
			response: &clijson.EntityJSON{PascalName: "StringValue"},
			schemas:  map[string]*clijson.SchemaJSON{"StringValue": {}},
			want:     false,
		},
		{
			name:     "fieldful object",
			response: &clijson.EntityJSON{PascalName: "FieldfulObject", IsObject: true},
			schemas:  map[string]*clijson.SchemaJSON{"FieldfulObject": {Fields: map[string]*clijson.SchemaFieldJSON{"name": {}}}},
			want:     false,
		},
		{
			name:     "unnamed response",
			response: &clijson.EntityJSON{IsObject: true},
			schemas:  map[string]*clijson.SchemaJSON{"": {}},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := &clijson.CommandsBlock{
				Services: []*clijson.ServiceJSON{{
					Methods: []*clijson.MethodJSON{{Response: tt.response}},
				}},
			}
			normalizeFieldlessResponses(commands, tt.schemas)
			transformed := fromContract(commands)
			if got := transformed.Services[0].Methods[0].Response.IsEmptyResponse; got != tt.want {
				t.Fatalf("IsEmptyResponse = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeFieldlessResponsesHandlesMissingNodes(t *testing.T) {
	commands := &clijson.CommandsBlock{
		Services: []*clijson.ServiceJSON{
			nil,
			{Methods: []*clijson.MethodJSON{nil, {}}},
		},
	}
	normalizeFieldlessResponses(commands, nil)
}

func TestFieldJSONIsCLIRequestField(t *testing.T) {
	tests := []struct {
		name  string
		field FieldJSON
		want  bool
	}{
		{name: "ordinary field", field: FieldJSON{}, want: true},
		{name: "output-only", field: FieldJSON{IsOutputOnly: true}, want: false},
		{name: "computed", field: FieldJSON{IsComputed: true}, want: false},
		{
			name:  "computed output-only request-body",
			field: FieldJSON{IsComputed: true, IsOutputOnly: true, IsRequestBodyField: true},
			want:  false,
		},
		{
			name:  "computed request-body",
			field: FieldJSON{IsComputed: true, IsRequestBodyField: true},
			want:  true,
		},
		{
			name:  "computed path",
			field: FieldJSON{IsComputed: true, IsPath: true, IsRequestBodyField: true},
			want:  false,
		},
		{
			name:  "computed query",
			field: FieldJSON{IsComputed: true, IsQuery: true, IsRequestBodyField: true},
			want:  false,
		},
		{
			name:  "computed non-body",
			field: FieldJSON{IsComputed: true, IsRequestBodyField: false},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.field.IsCLIRequestField(); got != tt.want {
				t.Fatalf("IsCLIRequestField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromContractPreservesRequestBodyFieldIdentity(t *testing.T) {
	requestBody := &clijson.FieldJSON{
		Name:               "payload",
		IsComputed:         true,
		IsRequestBodyField: true,
		Entity:             &clijson.EntityJSON{PascalName: "Payload", IsObject: true},
	}
	commands := fromContract(&clijson.CommandsBlock{
		Services: []*clijson.ServiceJSON{{
			ID:   "widgets",
			Name: "widgets",
			Methods: []*clijson.MethodJSON{{
				Name:             "update",
				RequestBodyField: &clijson.FieldJSON{Name: "payload"},
				AllFields:        []*clijson.FieldJSON{requestBody},
			}},
		}},
	})
	if err := commands.Resolve(); err != nil {
		t.Fatalf("Resolve(): %v", err)
	}
	method := commands.Services[0].Methods[0]
	if method.RequestBodyField != method.AllFields[0] {
		t.Fatal("RequestBodyField does not point to its AllFields entry")
	}
	if !method.RequestBodyField.IsCLIRequestField() {
		t.Fatal("computed request-body field should be a CLI request field")
	}
}

func TestServiceTemplateUsesCLIRequestFieldClassification(t *testing.T) {
	stringEntity := &EntityJSON{IsString: true}
	durationEntity := &EntityJSON{IsDuration: true}
	payload := &FieldJSON{
		Name:               "payload",
		IsComputed:         true,
		IsRequestBodyField: true,
		Entity:             &EntityJSON{PascalName: "Payload", IsObject: true},
	}
	method := &MethodJSON{
		Name:             "update",
		Request:          &EntityJSON{PascalName: "UpdateRequest", IsObject: true},
		Response:         &EntityJSON{PascalName: "Widget", IsObject: true},
		RequestBodyField: payload,
		AllFields: []*FieldJSON{
			{Name: "normal", Entity: durationEntity},
			{Name: "output-only", IsOutputOnly: true, Entity: stringEntity},
			{Name: "computed", IsComputed: true, Entity: stringEntity},
			{Name: "computed-body", IsComputed: true, IsRequestBodyField: true, Entity: durationEntity},
			{Name: "computed-path", IsComputed: true, IsPath: true, IsRequestBodyField: true, Entity: durationEntity},
			{Name: "computed-query", IsComputed: true, IsQuery: true, IsRequestBodyField: true, Entity: durationEntity},
			payload,
		},
	}
	service := &ServiceJSON{
		Name:    "widgets",
		Package: &PackageRef{Name: "compute"},
		Methods: []*MethodJSON{method},
	}

	var out strings.Builder
	if err := parseTemplate("service", "templates/service.go.tmpl").ExecuteTemplate(&out, "service", service); err != nil {
		t.Fatalf("render service: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{
		`cmd.Flags().StringVar(&normalParam, "normal"`,
		`cmd.Flags().StringVar(&computedBodyParam, "computed-body"`,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered service does not contain %q", want)
		}
	}
	for _, notWant := range []string{
		`, "output-only"`,
		`, "computed"`,
		`, "computed-path"`,
		`, "computed-query"`,
		"updateComputedPathParam",
		"updateComputedQueryParam",
	} {
		if strings.Contains(rendered, notWant) {
			t.Errorf("rendered service unexpectedly contains %q", notWant)
		}
	}
}

func TestServiceTemplateDistinguishesEmptyResponseArity(t *testing.T) {
	for _, tc := range []struct {
		name         string
		fieldless    bool
		wantCallText string
	}{
		{name: "explicit error-only", wantCallText: "err ="},
		{name: "named fieldless", fieldless: true, wantCallText: "_, err ="},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &ServiceJSON{
				Name:    "widgets",
				Package: &PackageRef{Name: "compute"},
				Methods: []*MethodJSON{{
					Name:                 "delete",
					IsLegacyEmptyRequest: true,
					Response:             &EntityJSON{IsEmptyResponse: true, IsFieldlessResponse: tc.fieldless},
				}},
			}
			var out strings.Builder
			if err := parseTemplate("service", "templates/service.go.tmpl").ExecuteTemplate(&out, "service", service); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), tc.wantCallText) {
				t.Fatalf("rendered call does not contain %q", tc.wantCallText)
			}
		})
	}
}
