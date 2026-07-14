package main

import (
	"strings"
	"testing"
)

// renderSkeletonService renders service.go.tmpl for a service whose methods
// exercise the --generate-skeleton wiring, and returns the generated source.
func renderSkeletonService(t *testing.T) string {
	t.Helper()

	svc := &ServiceJSON{
		Name:      "TestService",
		Package:   &PackageRef{Name: "testpkg"},
		DocsGroup: "testgroup",
		Methods: []*MethodJSON{
			{
				// JSON command whose body is the whole request.
				Name:       "Create",
				CanUseJson: true,
				Request:    &EntityJSON{PascalName: "CreateRequest"},
				Response:   &EntityJSON{IsEmptyResponse: true},
			},
			{
				// JSON command whose body is a nested request field.
				Name:             "Update",
				CanUseJson:       true,
				Request:          &EntityJSON{PascalName: "UpdateRequest"},
				RequestBodyField: &FieldJSON{Name: "resource", Entity: &EntityJSON{PascalName: "Resource", IsObject: true}},
				AllFields:        []*FieldJSON{{Name: "resource", IsRequestBodyField: true, Entity: &EntityJSON{PascalName: "Resource", IsObject: true}}},
				Response:         &EntityJSON{IsEmptyResponse: true},
			},
			{
				// Non-JSON command must not get the flag.
				Name:       "Get",
				CanUseJson: false,
				Request:    &EntityJSON{PascalName: "GetRequest"},
				Response:   &EntityJSON{IsEmptyResponse: true},
			},
		},
	}
	batch := &CommandsBlock{Services: []*ServiceJSON{svc}}
	if err := batch.Resolve(); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	tmpl := parseTemplate("service", "templates/service.go.tmpl")
	var sb strings.Builder
	if err := tmpl.ExecuteTemplate(&sb, "service.go.tmpl", svc); err != nil {
		t.Fatalf("render: %v", err)
	}
	return sb.String()
}

func TestTemplateWiresGenerateSkeleton(t *testing.T) {
	src := renderSkeletonService(t)

	// Plain request body: skeleton targets the request struct itself.
	if !strings.Contains(src, "root.RegisterGenerateSkeleton(cmd, &createReq)") {
		t.Errorf("expected skeleton call for Create targeting &createReq\n%s", src)
	}

	// Request-body-field command: skeleton targets the body field, matching the
	// --json unmarshal target.
	if !strings.Contains(src, "root.RegisterGenerateSkeleton(cmd, &updateReq.Resource)") {
		t.Errorf("expected skeleton call for Update targeting &updateReq.Resource\n%s", src)
	}

	// The skeleton target must match the --json unmarshal target for that method.
	if !strings.Contains(src, "updateJson.Unmarshal(&updateReq.Resource)") {
		t.Errorf("expected --json unmarshal into &updateReq.Resource\n%s", src)
	}
}

func TestTemplateSkipsGenerateSkeletonForNonJSON(t *testing.T) {
	src := renderSkeletonService(t)

	// Get does not accept --json, so it gets no skeleton flag. Exactly the two
	// JSON commands (Create, Update) are wired; count so a rename can't make the
	// absence check pass vacuously.
	if got := strings.Count(src, "root.RegisterGenerateSkeleton("); got != 2 {
		t.Errorf("expected 2 skeleton calls (Create, Update), got %d\n%s", got, src)
	}
	if strings.Contains(src, "root.RegisterGenerateSkeleton(cmd, &getReq)") {
		t.Errorf("did not expect a skeleton call for the non-JSON Get command\n%s", src)
	}
}
