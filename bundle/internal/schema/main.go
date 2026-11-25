package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/internal/clijson"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/jsonschema"
)

// interpolationPattern builds a JSON Schema regex for ${prefix.path...} references.
// Path segments use [dynvar.BaseVarDef]; unlike the runtime matcher in ref.go, this
// requires a fixed prefix (var, resources, ...) and at least one ".segment" after it.
func interpolationPattern(s string) string {
	return fmt.Sprintf(`\$\{(%s(\.%s(\[[0-9]+\])*)+)\}`, s, dynvar.BaseVarDef)
}

func addInterpolationPatterns(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	if typ == reflect.TypeFor[config.Root]() || typ == reflect.TypeFor[variable.Variable]() {
		return s
	}

	// The variables block in a target override allows for directly specifying
	// the value of the variable.
	if typ == reflect.TypeFor[variable.TargetVariable]() {
		return jsonschema.Schema{
			AnyOf: []jsonschema.Schema{
				// We keep the original schema so that autocomplete suggestions
				// continue to work.
				s,
				// All values are valid for a variable value, be it primitive types
				// like string/bool or complex ones like objects/arrays. Thus we override
				// the schema to allow all valid JSON values.
				{},
			},
		}
	}

	// Allows using variables in enum fields
	if s.Type == jsonschema.StringType && s.Enum != nil {
		return jsonschema.Schema{
			OneOf: []jsonschema.Schema{
				s,
				{
					Type:    jsonschema.StringType,
					Pattern: interpolationPattern("var"),
				},
			},
		}
	}

	switch s.Type {
	case jsonschema.ArrayType, jsonschema.ObjectType:
		// arrays and objects can have complex variable values specified.
		return jsonschema.Schema{
			// OneOf is used because we don't expect more than 1 match and schema-based auto-complete works better with OneOf
			OneOf: []jsonschema.Schema{
				s,
				{
					Type:    jsonschema.StringType,
					Pattern: interpolationPattern("var"),
				},
			},
		}
	case jsonschema.IntegerType, jsonschema.NumberType, jsonschema.BooleanType:
		// primitives can have variable values, or references like ${bundle.xyz}
		// or ${workspace.xyz}
		return jsonschema.Schema{
			OneOf: []jsonschema.Schema{
				s,
				{Type: jsonschema.StringType, Pattern: interpolationPattern("resources")},
				{Type: jsonschema.StringType, Pattern: interpolationPattern("bundle")},
				{Type: jsonschema.StringType, Pattern: interpolationPattern("workspace")},
				{Type: jsonschema.StringType, Pattern: interpolationPattern("artifacts")},
				{Type: jsonschema.StringType, Pattern: interpolationPattern("var")},
			},
		}
	default:
		return s
	}
}

// resourceSchemaTransforms returns a transform that applies the schema
// transforms resource types declare (see [resources.SchemaTransformer]). Where
// several resource types declare a transform for the same Go type, they run in
// the canonical order of resource types.
func resourceSchemaTransforms() func(reflect.Type, jsonschema.Schema) jsonschema.Schema {
	byType := map[reflect.Type][]func(jsonschema.Schema) jsonschema.Schema{}
	for _, d := range config.ResourceTypesImplementing[resources.SchemaTransformer]() {
		for typ, fn := range d.SchemaTransforms() {
			byType[typ] = append(byType[typ], fn)
		}
	}
	return func(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
		for _, fn := range byType[typ] {
			s = fn(s)
		}
		return s
	}
}

func removeOutputOnlyFields(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	// Only process object types with properties
	if s.Type != jsonschema.ObjectType || s.Properties == nil {
		return s
	}

	// Collect property names to remove
	var toRemove []string
	for name, prop := range s.Properties {
		// Check if this property is marked as output-only via FieldBehaviors
		if prop.FieldBehaviors != nil {
			if slices.Contains(prop.FieldBehaviors, "OUTPUT_ONLY") {
				toRemove = append(toRemove, name)
			}
		}
	}

	// Remove output-only properties
	for _, name := range toRemove {
		delete(s.Properties, name)

		// Also remove from required list if present
		var newRequired []string
		for _, r := range s.Required {
			if r != name {
				newRequired = append(newRequired, r)
			}
		}
		s.Required = newRequired
	}

	return s
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run main.go <work-dir> <output-file> <cli-json> [--docs]")
		os.Exit(1)
	}

	// Directory with the annotations file
	workdir := os.Args[1]
	// Output file, where the generated JSON schema will be written to.
	outputFile := os.Args[2]
	// The .codegen/cli.json spec. Its schema graph carries the descriptions,
	// enums and field behaviors the CLI reflects onto its config types.
	cliJSONFile := os.Args[3]

	// When --docs is passed, skip interpolation patterns and add sinceVersion annotations.
	// This generates a schema optimized for documentation.
	docsMode := len(os.Args) >= 5 && os.Args[4] == "--docs"

	generateSchema(workdir, outputFile, cliJSONFile, docsMode)
}

// annotationsDirName is the directory of annotation files in the generator's
// work directory; see annotations_file.go for its layout.
const annotationsDirName = "annotations"

// configTypeGraph builds the type graph for the bundle config root with the
// schema generator's structural field prunes applied, so the annotations file
// only documents fields the generated schema actually contains. Both the
// generator and the detached-annotation test use it to stay in lockstep.
func configTypeGraph() (*typeGraph, error) {
	return newTypeGraph(reflect.TypeFor[config.Root](), resourceSchemaTransforms())
}

func generateSchema(workdir, outputFile, cliJSONFile string, docsMode bool) {
	annotationsDir := filepath.Join(workdir, annotationsDirName)

	// The cli.json schema graph is keyed by SDK type name (e.g.
	// "jobs.JobSettings"); the annotation parser matches Go SDK types against
	// those keys directly.
	doc, err := clijson.Parse(cliJSONFile)
	if err != nil {
		log.Fatal(err)
	}
	if len(doc.Schemas) == 0 {
		log.Fatalf("no schemas found in %s", cliJSONFile)
	}

	extracted, err := newParser(doc.Schemas).extractAnnotations(reflect.TypeFor[config.Root]())
	if err != nil {
		log.Fatal(err)
	}

	graph, err := configTypeGraph()
	if err != nil {
		log.Fatal(err)
	}

	// Both hand-written layers share one format and are merged into one set;
	// the split between them is recomputed on save.
	fromFile, unknown, err := loadAnnotationsDir(annotationsDir, handSuffix, graph)
	if err != nil {
		log.Fatal(err)
	}
	overrides, unknownOverrides, err := loadAnnotationsDir(annotationsDir, overridesSuffix, graph)
	if err != nil {
		log.Fatal(err)
	}
	for _, k := range append(unknown, unknownOverrides...) {
		fmt.Printf("Dropping annotation at `%s`: no matching field in the bundle configuration\n", k)
	}
	fromFile, err = mergeAnnotationFiles(fromFile, overrides)
	if err != nil {
		log.Fatal(err)
	}

	// fromFile feeds both the merge in newAnnotationHandler and the
	// annotations-file rewrite in syncAnnotationFiles, so strip placeholders
	// first: they belong in the generated file and would otherwise shadow an
	// upstream description.
	dropPlaceholders(fromFile)

	a, err := newAnnotationHandler(extracted, fromFile)
	if err != nil {
		log.Fatal(err)
	}

	transforms := []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		resourceSchemaTransforms(),
		a.addAnnotations,
		removeOutputOnlyFields,
	}
	if !docsMode {
		transforms = append(transforms, addInterpolationPatterns)
	}

	// Generate the JSON schema from the bundle Go struct.
	s, err := jsonschema.FromType(reflect.TypeFor[config.Root](), transforms)

	// AdditionalProperties is set to an empty schema to allow non-typed keys used as yaml-anchors
	// Example:
	// some_anchor: &some_anchor
	//   file_path: /some/path/
	// workspace:
	//   <<: *some_anchor
	s.AdditionalProperties = jsonschema.Schema{}

	if err != nil {
		log.Fatal(err)
	}

	// Rewrite the hand-written files in canonical form and the generated
	// files with the fields that still lack documentation.
	err = a.syncAnnotationFiles(annotationsDir, graph)
	if err != nil {
		log.Fatal(err)
	}

	// In docs mode, add sinceVersion annotations by analyzing git history.
	if docsMode {
		sinceVersions, err := computeSinceVersions()
		if err != nil {
			fmt.Printf("Warning: could not compute sinceVersion annotations: %v\n", err)
		} else {
			addSinceVersionToSchema(&s, sinceVersions)
		}
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	// Write the schema descriptions to the output file.
	err = os.WriteFile(outputFile, b, 0o644)
	if err != nil {
		log.Fatal(err)
	}
}
