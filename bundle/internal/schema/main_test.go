package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cliJSONPath = "../../../.codegen/cli.json"

func TestJobRunIdempotencyTokenIsNotInSchema(t *testing.T) {
	s := jsonschema.Schema{
		Properties: map[string]*jsonschema.Schema{
			"idempotency_token": {},
			"job_id":            {},
		},
	}

	s = removeJobsFields(reflect.TypeFor[resources.JobRun](), s)

	assert.NotContains(t, s.Properties, "idempotency_token")
	assert.Contains(t, s.Properties, "job_id")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Close()
}

// copyDir copies the regular files of src into dst.
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		err = copyFile(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}

// regenerateAnnotationFiles regenerates the annotations directory from the
// committed one into a temp dir and returns, per file, the tree paths that
// were added and the paths that were changed or removed. Files present on only
// one side are reported as a whole.
func regenerateAnnotationFiles(t *testing.T) (added, changed map[string][]string) {
	workdir := t.TempDir()
	dir := filepath.Join(workdir, annotationsDirName)
	require.NoError(t, os.Mkdir(dir, 0o755))
	require.NoError(t, copyDir(annotationsDirName, dir))

	generateSchema(workdir, filepath.Join(t.TempDir(), "schema.json"), cliJSONPath, false)

	added = map[string][]string{}
	changed = map[string][]string{}
	names := map[string]bool{}
	for _, d := range []string{annotationsDirName, dir} {
		entries, err := os.ReadDir(d)
		require.NoError(t, err)
		for _, e := range entries {
			names[e.Name()] = true
		}
	}
	for name := range names {
		originalFile, err := os.ReadFile(filepath.Join(annotationsDirName, name))
		if errors.Is(err, os.ErrNotExist) {
			added[name] = append(added[name], "(new file)")
			continue
		}
		require.NoError(t, err)
		currentFile, err := os.ReadFile(filepath.Join(dir, name))
		if errors.Is(err, os.ErrNotExist) {
			changed[name] = append(changed[name], "(file removed)")
			continue
		}
		require.NoError(t, err)
		if name == "README.md" {
			continue
		}
		original, err := yamlloader.LoadYAML("", bytes.NewBuffer(originalFile))
		require.NoError(t, err)
		current, err := yamlloader.LoadYAML("", bytes.NewBuffer(currentFile))
		require.NoError(t, err)

		// VisitDelete/VisitUpdate must be set or Override panics on any change.
		_, err = merge.Override(original, current, merge.OverrideVisitor{
			VisitInsert: func(basePath dyn.Path, right dyn.Value) (dyn.Value, error) {
				added[name] = append(added[name], basePath.String())
				return right, nil
			},
			VisitDelete: func(basePath dyn.Path, left dyn.Value) error {
				changed[name] = append(changed[name], basePath.String())
				return nil
			},
			VisitUpdate: func(basePath dyn.Path, left, right dyn.Value) (dyn.Value, error) {
				changed[name] = append(changed[name], basePath.String())
				return right, nil
			},
		})
		require.NoError(t, err)
	}
	return added, changed
}

// Checks whether descriptions are added for new config fields.
// If this test fails:
//  1. run `./task generate-schema` from the repository root; new undocumented
//     fields appear as PLACEHOLDER entries in annotations/*.generated.yml
//  2. document them in the matching annotations/*.yml at the same tree
//     position if possible
//  3. run `./task generate-schema` again to regenerate the schema with the
//     new descriptions and drop the entries from the generated files
func TestRequiredAnnotationsForNewFields(t *testing.T) {
	added, changed := regenerateAnnotationFiles(t)

	var missing, stale []string
	for name, paths := range added {
		for _, p := range paths {
			if strings.HasSuffix(name, generatedSuffix+yamlExt) {
				missing = append(missing, name+": "+p)
			} else {
				stale = append(stale, name+": "+p)
			}
		}
	}
	for name, paths := range changed {
		for _, p := range paths {
			stale = append(stale, name+": "+p)
		}
	}
	slices.Sort(missing)
	slices.Sort(stale)
	assert.Empty(t, missing, "Missing JSON-schema descriptions for new config fields; run `./task generate-schema` and document them in bundle/internal/schema/annotations/:\n%s", strings.Join(missing, "\n"))
	assert.Empty(t, stale, "bundle/internal/schema/annotations/ is out of sync; run `./task generate-schema` and commit the result:\n%s", strings.Join(stale, "\n"))
}

// Checks that the annotations file only contains entries that match the
// current bundle configuration structure.
func TestNoDetachedAnnotations(t *testing.T) {
	g, err := configTypeGraph()
	require.NoError(t, err)

	for _, suffix := range []string{handSuffix, overridesSuffix, generatedSuffix} {
		_, unknown, err := loadAnnotationsDir(annotationsDirName, suffix, g)
		require.NoError(t, err)
		assert.Empty(t, unknown, "Detached annotations found in *%s.yml; run `./task generate-schema` to drop them", suffix)
	}
}
