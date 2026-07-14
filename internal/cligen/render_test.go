package main

import "testing"

// TestTemplatesParse ensures every copied template parses against helperFuncs.
// text/template requires every called function to be registered at parse time,
// so this fails fast if a template references a helper missing from helperFuncs.
func TestTemplatesParse(t *testing.T) {
	cases := []struct{ name, path string }{
		{"service", "templates/service.go.tmpl"},
		{"cmds-workspace.go.tmpl", "templates/cmds-workspace.go.tmpl"},
		{"groups-workspace.go.tmpl", "templates/groups-workspace.go.tmpl"},
		{"cmds-account.go.tmpl", "templates/cmds-account.go.tmpl"},
		{"groups-account.go.tmpl", "templates/groups-account.go.tmpl"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("parsing %s panicked: %v", c.path, r)
				}
			}()
			_ = parseTemplate(c.name, c.path)
		})
	}
}
