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

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/bundle/internal/sdkmeta"
	"github.com/databricks/cli/internal/clijson"
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
func renameDefinitions(s *jsonschema.Schema, meta *sdkmeta.Meta, schemas map[string]*clijson.SchemaJSON, docsMode bool) {
	names := map[string]string{}
	order := map[string]int{}
	for t := range meta.Types() {
		if name, ok := meta.DefinitionName(t); ok {
			names[jsonschemaTypePath(t)] = name
		}
		if i, ok := meta.Order(t); ok {
			order[jsonschemaTypePath(t)] = i
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

	// Merge in walk order so that the type annotations.yml documents (the first
	// occurrence in the configuration) wins.
	keys := slices.Sorted(maps.Keys(flat))
	pos := func(k string) int {
		if i, ok := order[k]; ok {
			return i
		}
		return len(order)
	}
	slices.SortStableFunc(keys, func(a, b string) int { return pos(a) - pos(b) })
	renamed := map[string]jsonschema.Schema{}
	for _, k := range keys {
		def := flat[k]
		rewriteRefs(&def, rename)
		nk := rename(k)
		if prev, ok := renamed[nk]; ok {
			def = mergeDefinitions(nk, prev, def)
		}
		renamed[nk] = def
	}
	rewriteRefs(s, rename)
	applyFieldRefs(renamed, schemas, docsMode)
	pruneToSpec(renamed, schemas)
	dropUnreachableAPIDefinitions(s, renamed)
	s.Definitions = nestDefinitions(renamed)
}

// pruneToSpec removes properties of API definitions that their cli.json
// schema does not define. A type sdk-go unified from several previous types
// carries the union of their fields until here (see removeFieldsMissingFromSpec).
func pruneToSpec(defs map[string]jsonschema.Schema, schemas map[string]*clijson.SchemaJSON) {
	for k, def := range defs {
		name, ok := strings.CutPrefix(k, sdkmeta.LegacyPkgPrefix)
		if !ok {
			continue
		}
		spec := schemas[name]
		obj := objectSchema(&def)
		if spec == nil || spec.Fields == nil || obj == nil {
			continue
		}
		for prop := range obj.Properties {
			if spec.Fields[prop] == nil {
				delete(obj.Properties, prop)
				obj.Required = slices.DeleteFunc(obj.Required, func(r string) bool { return r == prop })
			}
		}
		defs[k] = def
	}
}

// applyFieldRefs points fields whose type sdk-go changed back at the
// definitions they referred to before (see sdkmeta.FieldRef): enums that
// became plain strings get their enum definition from cli.json, and fields of
// unified messages get a copy of the unified definition under the old name.
func applyFieldRefs(defs map[string]jsonschema.Schema, schemas map[string]*clijson.SchemaJSON, docsMode bool) {
	for _, k := range slices.Sorted(maps.Keys(defs)) {
		name, ok := strings.CutPrefix(k, sdkmeta.LegacyPkgPrefix)
		if !ok {
			continue
		}
		def := defs[k]
		obj := objectSchema(&def)
		if obj == nil {
			continue
		}
		for prop, ps := range obj.Properties {
			ref, ok := sdkmeta.FieldRef(name, prop)
			if !ok || ps.Reference == nil {
				continue
			}
			target := strings.TrimPrefix(*ps.Reference, "#/$defs/")
			newTarget := ensureDefinition(defs, target, sdkmeta.LegacyPkgPrefix+ref, schemas[ref], docsMode)
			*ps.Reference = "#/$defs/" + newTarget
		}
		defs[k] = def
	}
}

// ensureDefinition makes a definition named newBase from the definition target
// refers to (through slice/ and map/ wrappers, which are recreated), and
// returns the new key replacing target.
func ensureDefinition(defs map[string]jsonschema.Schema, target, newBase string, spec *clijson.SchemaJSON, docsMode bool) string {
	head, rest, ok := strings.Cut(target, "/")
	if ok && (head == "slice" || head == "map") {
		inner := ensureDefinition(defs, rest, newBase, spec, docsMode)
		key := head + "/" + inner
		if _, ok := defs[key]; !ok {
			wrapper := cloneSchema(defs[target])
			ref := "#/$defs/" + inner
			if len(wrapper.OneOf) == 0 {
				switch head {
				case "slice":
					wrapper.Items = &jsonschema.Schema{Reference: &ref}
				case "map":
					wrapper.AdditionalProperties = &jsonschema.Schema{Reference: &ref}
				}
			} else {
				switch head {
				case "slice":
					wrapper.OneOf[0].Items = &jsonschema.Schema{Reference: &ref}
				case "map":
					wrapper.OneOf[0].AdditionalProperties = &jsonschema.Schema{Reference: &ref}
				}
			}
			defs[key] = wrapper
		}
		return key
	}
	if _, ok := defs[newBase]; ok {
		return newBase
	}
	if target == "string" && spec != nil && len(spec.Enum) > 0 {
		defs[newBase] = enumDefinition(spec, docsMode)
		return newBase
	}
	def := cloneSchema(defs[target])
	if o := objectSchema(&def); o != nil && spec != nil {
		o.Description = spec.Description
		name := strings.TrimPrefix(newBase, sdkmeta.LegacyPkgPrefix)
		o.Required = nil
		for _, r := range sdkmeta.RequiredFields(name) {
			if _, ok := o.Properties[r]; ok {
				o.Required = append(o.Required, r)
			}
		}
		for prop, ps := range o.Properties {
			if f := spec.Fields[prop]; f != nil && f.Description != "" {
				stage, _ := clijson.ParseLaunchStage(f.LaunchStage)
				ps.Description = prefixWithPreviewTag(f.Description, annotation.PreviewTag(stage))
			}
		}
	}
	defs[newBase] = def
	return newBase
}

// enumDefinition builds the definition of an enum from its cli.json schema,
// the way the generator does for enum types.
func enumDefinition(spec *clijson.SchemaJSON, docsMode bool) jsonschema.Schema {
	s := jsonschema.Schema{Type: jsonschema.StringType}
	stages, err := notableEnumLaunchStages(spec.EnumLaunchStages)
	if err != nil {
		log.Fatal(err)
	}
	assignAnnotation(&s, annotation.Descriptor{
		Description:      spec.Description,
		Enum:             enumValues(spec.Enum),
		EnumLaunchStages: stages,
		EnumDescriptions: nonEmptyEnumDescriptions(spec.EnumDescriptions),
	})
	if docsMode {
		return s
	}
	return addInterpolationPatterns(reflect.TypeFor[string](), s)
}

func cloneSchema(s jsonschema.Schema) jsonschema.Schema {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	var out jsonschema.Schema
	if err := json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}

// jsonschemaTypePath is the $defs key jsonschema.FromType uses for t.
func jsonschemaTypePath(t reflect.Type) string {
	name, _, _ := strings.Cut(t.Name(), "[")
	return t.PkgPath() + "." + name
}

func flattenDefinitions(prefix string, m map[string]any, out map[string]jsonschema.Schema) {
	for k, v := range m {
		var p string
		if prefix != "" {
			p = prefix + "/" + k
		} else {
			p = k
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

// dropUnreachableAPIDefinitions removes definitions of sdk-go types that are
// not referenced from the schema, i.e. types only used by fields pruned
// because cli.json does not define them. Other unreferenced definitions are
// kept, as the generator always emitted them.
func dropUnreachableAPIDefinitions(root *jsonschema.Schema, defs map[string]jsonschema.Schema) {
	reached := map[string]bool{}
	var visit func(s *jsonschema.Schema)
	visit = func(s *jsonschema.Schema) {
		if s == nil {
			return
		}
		if s.Reference != nil {
			k := strings.TrimPrefix(*s.Reference, "#/$defs/")
			if !reached[k] {
				reached[k] = true
				def := defs[k]
				visit(&def)
			}
		}
		for _, p := range s.Properties {
			visit(p)
		}
		visit(s.Items)
		if ap, ok := s.AdditionalProperties.(*jsonschema.Schema); ok {
			visit(ap)
		}
		for i := range s.AnyOf {
			visit(&s.AnyOf[i])
		}
		for i := range s.OneOf {
			visit(&s.OneOf[i])
		}
	}
	visit(root)
	for k := range defs {
		if !reached[k] && (strings.Contains(k, "github.com/databricks/cli/libs/sdktypes/") || strings.Contains(k, "github.com/databricks/sdk-go/")) {
			delete(defs, k)
		}
	}
}
