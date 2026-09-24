package main

import (
	"encoding/json"
	"errors"
	goformat "go/format"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/clijson"
)

// cliJSONPath is the checked-in contract instance, relative to this package.
const cliJSONPath = "../../.codegen/cli.json"

// fullDoc decodes every block of cli.json that the CLI cares about. The schemas
// block is consumed by the Python direct-engine generators; here we only need
// enough of it to assert structural invariants (keys + field refs).
type fullDoc struct {
	Schemas  map[string]schemaDoc `json:"schemas"`
	Commands *CommandsBlock       `json:"commands"`
}

type schemaDoc struct {
	Fields map[string]struct {
		Ref string `json:"ref"`
	} `json:"fields"`
}

var shaRE = regexp.MustCompile(`^[0-9a-f]{40}$`)

// TestCliJSONIsInterpretable loads the committed cli.json and asserts the CLI
// can interpret it: it decodes into the CLI's model, its cross-references
// resolve, its contents satisfy the invariants the renderer relies on, and
// every service + batch file renders to syntactically valid Go. This is the
// CLI-repo-side confirmation that the cli.json spec is consumable.
func TestCliJSONIsInterpretable(t *testing.T) {
	raw, err := os.ReadFile(cliJSONPath)
	if err != nil {
		t.Fatalf("read %s: %v", cliJSONPath, err)
	}

	var doc fullDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode cli.json: %v", err)
	}

	// The spec SHA is not carried in cli.json metadata; genkit writes it to the
	// sibling _openapi_sha file, the same convention every other SDK target uses.
	shaBytes, err := os.ReadFile("../../.codegen/_openapi_sha")
	if err != nil {
		t.Fatalf("read _openapi_sha: %v", err)
	}
	if sha := strings.TrimSpace(string(shaBytes)); !shaRE.MatchString(sha) {
		t.Errorf("_openapi_sha = %q, want a 40-char sha", sha)
	}

	// --- schemas block: non-empty, and every field ref points at a known schema ---
	if len(doc.Schemas) == 0 {
		t.Fatal("schemas block is empty")
	}
	for name, s := range doc.Schemas {
		for fname, f := range s.Fields {
			if f.Ref != "" {
				if _, ok := doc.Schemas[f.Ref]; !ok {
					t.Errorf("schema %s.%s: ref %q points at a missing schema", name, fname, f.Ref)
				}
			}
		}
	}

	// --- commands block: present, resolvable, internally consistent ---
	if doc.Commands == nil {
		t.Fatal("commands block is nil")
	}
	cmds := doc.Commands
	if err := cmds.Resolve(); err != nil {
		t.Fatalf("resolve commands: %v", err)
	}
	if len(cmds.Services) == 0 {
		t.Fatal("commands has no services")
	}

	// Two services rendering to the same path would silently overwrite each
	// other's file (Generate writes them in order, last wins).
	filenames := map[string]string{}
	for _, s := range cmds.Services {
		fn := serviceFilename(s)
		if prev, ok := filenames[fn]; ok {
			t.Errorf("services %s and %s both render to %s", prev, s.Name, fn)
		}
		filenames[fn] = s.Name
	}

	for _, s := range cmds.Services {
		if s.Name == "" {
			t.Errorf("service %q has empty name", s.ID)
		}
		if s.Package == nil || s.Package.Name == "" {
			t.Errorf("service %s has no package name", s.Name)
		}
		if s.ParentServiceID != "" && s.ParentService == nil {
			t.Errorf("service %s: parent_service_id %q did not resolve", s.Name, s.ParentServiceID)
		}
		if len(s.Subservices) != len(s.SubserviceIDs) {
			t.Errorf("service %s: %d subservice ids but %d resolved", s.Name, len(s.SubserviceIDs), len(s.Subservices))
		}
		for _, m := range s.Methods {
			if m.Name == "" {
				t.Errorf("service %s has a method with empty name", s.Name)
			}
		}
	}

	// --- interpretation proof: every file renders to valid Go ---
	serviceTmpl := parseTemplate("service", "templates/service.go.tmpl")
	rendered := 0
	for _, svc := range cmds.Services {
		var sb strings.Builder
		err := serviceTmpl.ExecuteTemplate(&sb, "service.go.tmpl", svc)
		if errors.Is(err, ErrSkipThisFile) {
			continue
		}
		if err != nil {
			t.Errorf("render service %s: %v", svc.Name, err)
			continue
		}
		if _, err := goformat.Source([]byte(sb.String())); err != nil {
			t.Errorf("service %s rendered invalid Go: %v", svc.Name, err)
		}
		rendered++
	}
	if rendered == 0 {
		t.Fatal("no services rendered")
	}

	for _, bf := range []struct{ name, path string }{
		{"cmds-workspace.go.tmpl", "templates/cmds-workspace.go.tmpl"},
		{"groups-workspace.go.tmpl", "templates/groups-workspace.go.tmpl"},
		{"cmds-account.go.tmpl", "templates/cmds-account.go.tmpl"},
		{"groups-account.go.tmpl", "templates/groups-account.go.tmpl"},
	} {
		var sb strings.Builder
		tmpl := parseTemplate(bf.name, bf.path)
		if err := tmpl.ExecuteTemplate(&sb, bf.name, cmds); err != nil {
			t.Errorf("render %s: %v", bf.name, err)
			continue
		}
		if _, err := goformat.Source([]byte(sb.String())); err != nil {
			t.Errorf("%s rendered invalid Go: %v", bf.name, err)
		}
	}
}

func TestCheckedInCliJSONFieldClassification(t *testing.T) {
	doc, err := clijson.Parse(cliJSONPath)
	if err != nil {
		t.Fatalf("parse %s: %v", cliJSONPath, err)
	}
	commands := fromContract(doc.Commands)
	method := findMethod(t, commands, "Lakeview", "Create")
	want := map[string]bool{
		"parent_path":  true,
		"etag":         true,
		"dashboard_id": false,
	}
	for name, isCLIRequestField := range want {
		field := findField(t, method.AllFields, name)
		if got := field.IsCLIRequestField(); got != isCLIRequestField {
			t.Errorf("Lakeview.Create field %q IsCLIRequestField() = %v, want %v", name, got, isCLIRequestField)
		}
	}
}

func TestCheckedInCliJSONFieldlessResponses(t *testing.T) {
	doc, err := clijson.Parse(cliJSONPath)
	if err != nil {
		t.Fatalf("parse %s: %v", cliJSONPath, err)
	}
	fieldless := normalizeFieldlessResponses(doc.Commands, doc.Schemas)
	commands := fromContractWithFieldless(doc.Commands, fieldless)
	for _, target := range []struct {
		service string
		method  string
	}{
		{service: "AccountMetastoreAssignments", method: "Create"},
		{service: "AccountMetastoreAssignments", method: "Delete"},
		{service: "AiGateway", method: "DeleteMcpServiceUserMappedCredential"},
	} {
		t.Run(target.service+"."+target.method, func(t *testing.T) {
			method := findMethod(t, commands, target.service, target.method)
			if method.Response == nil || !method.Response.IsEmptyResponse || !method.Response.IsFieldlessResponse {
				t.Errorf("response is not normalized as a fieldless empty response: %#v", method.Response)
			}
		})
	}
	explicit := findMethod(t, commands, "Budgets", "Delete")
	if explicit.Response == nil || !explicit.Response.IsEmptyResponse || explicit.Response.IsFieldlessResponse {
		t.Errorf("explicit empty response was treated as fieldless: %#v", explicit.Response)
	}
}

func findMethod(t *testing.T, commands *CommandsBlock, serviceName, methodName string) *MethodJSON {
	t.Helper()
	for _, service := range commands.Services {
		if service.Name != serviceName {
			continue
		}
		for _, method := range service.Methods {
			if method.Name == methodName {
				return method
			}
		}
	}
	t.Fatalf("method %s.%s not found", serviceName, methodName)
	return nil
}

func findField(t *testing.T, fields []*FieldJSON, name string) *FieldJSON {
	t.Helper()
	for _, field := range fields {
		if field.Name == name {
			return field
		}
	}
	t.Fatalf("field %q not found", name)
	return nil
}
