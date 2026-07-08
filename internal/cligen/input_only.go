package main

import (
	"slices"
	"strings"

	"github.com/databricks/cli/internal/clijson"
)

// inputOnlyBehavior is the field-behavior token in cli.json that marks a field
// the server reads on writes but never returns on responses. The constant lives
// here rather than as an enum because cli.json carries Behaviors as []string;
// matching by value keeps the contract dependency one-way.
const inputOnlyBehavior = "INPUT_ONLY"

// populateInputOnlyPaths fills MethodJSON.InputOnlyPaths for every method
// whose response is a singleton message and that takes the normal sync render
// path (no pagination / LRO / wait / byte stream). The paths drive an
// inputonly.Strip call emitted by service.go.tmpl before cmdio.Render so the
// SDK transport struct's INPUT_ONLY fields don't leak into JSON output.
//
// Methods on the LRO / wait / pagination / byte-stream branches still render
// directly via cmdio.Render — extending those is a follow-up because each
// branch's rendered shape differs (operation type, iterator element, byte
// stream) and cli.json does not yet surface iterator element types.
func populateInputOnlyPaths(batch *CommandsBlock, schemas map[string]*clijson.SchemaJSON) {
	if len(schemas) == 0 {
		return
	}
	for _, s := range batch.Services {
		if s.Package == nil {
			continue
		}
		for _, m := range s.Methods {
			if !eligibleForInputOnly(m) {
				continue
			}
			rootName := s.Package.Name + "." + m.Response.PascalName
			m.InputOnlyPaths = inputOnlyPaths(schemas, rootName)
		}
	}
}

// eligibleForInputOnly picks the methods whose render site is the standard
// `cmdio.Render(ctx, response)` call in the method-call template.
func eligibleForInputOnly(m *MethodJSON) bool {
	if m.Response == nil || m.Response.PascalName == "" || m.Response.IsEmptyResponse {
		return false
	}
	if m.IsResponseByteStream {
		return false
	}
	if m.Pagination != nil || m.Wait != nil || m.LongRunningOperation != nil {
		return false
	}
	return true
}

// inputOnlyPaths walks the schema graph rooted at rootName and returns the
// sorted dotted JSON paths of every INPUT_ONLY field reachable via direct
// message-typed refs.
//
// Array and map element types are not followed: cli.json's SchemaFieldJSON
// carries one ref slot, populated for singleton message-typed fields only —
// `collaborators` (an array of CleanRoomCollaborator) has no ref, so a field
// like `invite_recipient_workspace_id` reachable only via that array is not
// caught here. A field that is itself INPUT_ONLY at the container level is
// still emitted as a single path; the runtime in libs/inputonly traverses
// arrays and maps transparently.
//
// Cycles in the schema graph are pruned via a per-descent set: the same
// schema can still be reached from multiple parent paths and emit different
// paths for each, but recursion never loops.
func inputOnlyPaths(schemas map[string]*clijson.SchemaJSON, rootName string) []string {
	var paths []string
	onPath := map[string]bool{}
	var walk func(prefix []string, schemaName string)
	walk = func(prefix []string, schemaName string) {
		s := schemas[schemaName]
		if s == nil || onPath[schemaName] {
			return
		}
		onPath[schemaName] = true
		defer delete(onPath, schemaName)
		names := make([]string, 0, len(s.Fields))
		for name := range s.Fields {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			f := s.Fields[name]
			childPath := append(slices.Clone(prefix), name)
			if slices.Contains(f.Behaviors, inputOnlyBehavior) {
				paths = append(paths, strings.Join(childPath, "."))
				continue
			}
			if f.Ref != "" {
				walk(childPath, f.Ref)
			}
		}
	}
	walk(nil, rootName)
	slices.Sort(paths)
	return paths
}
