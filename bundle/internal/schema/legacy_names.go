package main

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/internal/sdkmeta"
	"github.com/databricks/cli/libs/jsonschema"
)

// renameDefinitions renames the $defs of API types to the names the previous
// SDK types had (see sdkmeta) and rewrites every $ref accordingly, so that the
// published schema and the Python classes generated from it keep their names.
//
// Several sdk-go types can map to one previous name: sdk-go duplicates shared
// messages per API (jobs.AwsAttributes, clusters.AwsAttributes, ...) where the
// previous SDK had a single compute.AwsAttributes. Such definitions are merged;
// properties present in only some of them are kept.
func renameDefinitions(s *jsonschema.Schema, meta *sdkmeta.Meta) {
	names := map[string]string{}
	for t := range meta.Types() {
		if name, ok := meta.DefinitionName(t); ok {
			names[jsonschemaTypePath(t)] = name
		}
	}

	rename := func(p string) string {
		var prefix []string
		for {
			head, rest, ok := strings.Cut(p, "/")
			if !ok || (head != "slice" && head != "map") {
				break
			}
			prefix = append(prefix, head)
			p = rest
		}
		if n, ok := names[p]; ok {
			p = n
		}
		return path.Join(append(prefix, p)...)
	}

	flat := map[string]jsonschema.Schema{}
	flattenDefinitions("", s.Definitions, flat)

	renamed := map[string]jsonschema.Schema{}
	for _, k := range slices.Sorted(maps.Keys(flat)) {
		def := flat[k]
		rewriteRefs(&def, rename)
		nk := rename(k)
		if prev, ok := renamed[nk]; ok {
			def = mergeDefinitions(nk, prev, def)
		}
		renamed[nk] = def
	}
	rewriteRefs(s, rename)
	s.Definitions = nestDefinitions(renamed)
}

// jsonschemaTypePath is the $defs key jsonschema.FromType uses for t.
func jsonschemaTypePath(t reflect.Type) string {
	name, _, _ := strings.Cut(t.Name(), "[")
	return t.PkgPath() + "." + name
}

func flattenDefinitions(prefix string, m map[string]any, out map[string]jsonschema.Schema) {
	for k, v := range m {
		p := k
		if prefix != "" {
			p = prefix + "/" + k
		}
		switch x := v.(type) {
		case jsonschema.Schema:
			out[p] = x
		case map[string]any:
			flattenDefinitions(p, x, out)
		default:
			log.Fatalf("unexpected definition %s of type %T", p, v)
		}
	}
}

func nestDefinitions(flat map[string]jsonschema.Schema) map[string]any {
	res := map[string]any{}
	for k, v := range flat {
		parts := strings.Split(k, "/")
		cur := res
		for _, p := range parts[:len(parts)-1] {
			next, ok := cur[p].(map[string]any)
			if !ok {
				next = map[string]any{}
				cur[p] = next
			}
			cur = next
		}
		cur[parts[len(parts)-1]] = v
	}
	return res
}

func rewriteRefs(s *jsonschema.Schema, rename func(string) string) {
	if s == nil {
		return
	}
	if s.Reference != nil {
		ref := "#/$defs/" + rename(strings.TrimPrefix(*s.Reference, "#/$defs/"))
		s.Reference = &ref
	}
	for _, p := range s.Properties {
		rewriteRefs(p, rename)
	}
	rewriteRefs(s.Items, rename)
	switch ap := s.AdditionalProperties.(type) {
	case *jsonschema.Schema:
		rewriteRefs(ap, rename)
	case jsonschema.Schema:
		rewriteRefs(&ap, rename)
		s.AdditionalProperties = ap
	}
	for i := range s.AnyOf {
		rewriteRefs(&s.AnyOf[i], rename)
	}
	for i := range s.OneOf {
		rewriteRefs(&s.OneOf[i], rename)
	}
}

// mergeDefinitions combines two definitions that map to the same name.
func mergeDefinitions(name string, a, b jsonschema.Schema) jsonschema.Schema {
	if equalSchemas(a, b) {
		return a
	}
	ao, bo := objectSchema(&a), objectSchema(&b)
	if ao == nil || bo == nil {
		// Enums and other non-object definitions: keep the first one.
		return a
	}
	for k, v := range bo.Properties {
		if _, ok := ao.Properties[k]; !ok {
			ao.Properties[k] = v
		}
	}
	var required []string
	for _, r := range ao.Required {
		if slices.Contains(bo.Required, r) {
			required = append(required, r)
		}
	}
	ao.Required = required
	return a
}

// objectSchema returns the object schema of a definition, which may be wrapped
// in a oneOf that allows variable interpolation.
func objectSchema(s *jsonschema.Schema) *jsonschema.Schema {
	if s.Type == jsonschema.ObjectType && s.Properties != nil {
		return s
	}
	for i := range s.OneOf {
		if s.OneOf[i].Type == jsonschema.ObjectType && s.OneOf[i].Properties != nil {
			return &s.OneOf[i]
		}
	}
	return nil
}

func equalSchemas(a, b jsonschema.Schema) bool {
	ab, err := json.Marshal(a)
	if err != nil {
		panic(fmt.Sprintf("marshal: %v", err))
	}
	bb, err := json.Marshal(b)
	if err != nil {
		panic(fmt.Sprintf("marshal: %v", err))
	}
	return string(ab) == string(bb)
}
