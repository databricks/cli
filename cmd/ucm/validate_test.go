package ucm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmdUcm "github.com/databricks/cli/cmd/ucm"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runValidate invokes the cobra ucm-subtree in a temp cwd set to fixtureDir
// and returns stdout, diag-stream output (cmdio stderr), and whatever the
// Execute call returned.
func runValidate(t *testing.T, fixtureDir string) (string, string, error) {
	t.Helper()

	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(fixtureDir))
	t.Cleanup(func() { _ = os.Chdir(prev) })

	cmd := cmdUcm.New()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"validate"})

	ctx, diagOut := cmdio.NewTestContextWithStderr(context.Background())
	ctx = logdiag.InitContext(ctx)
	logdiag.SetRoot(ctx, fixtureDir)
	cmd.SetContext(ctx)

	err = cmd.Execute()
	return out.String(), diagOut.String() + errOut.String(), err
}

func TestCmd_Validate_ValidFixturePasses(t *testing.T) {
	stdout, stderr, err := runValidate(t, filepath.Join("testdata", "valid"))
	t.Logf("stdout=%q", stdout)
	t.Logf("stderr=%q", stderr)
	require.NoError(t, err)
	assert.Contains(t, stdout, "Validation OK!")
}

func TestCmd_Validate_MissingTagFixtureFails(t *testing.T) {
	_, _, err := runValidate(t, filepath.Join("testdata", "missing_tag"))
	require.Error(t, err)
}

func TestCmd_Validate_NestedFixturePasses(t *testing.T) {
	stdout, stderr, err := runValidate(t, filepath.Join("testdata", "nested"))
	t.Logf("stdout=%q", stdout)
	t.Logf("stderr=%q", stderr)
	require.NoError(t, err)
	assert.Contains(t, stdout, "Validation OK!")
}

func TestCmd_Validate_CollisionFixtureFails(t *testing.T) {
	_, stderr, err := runValidate(t, filepath.Join("testdata", "collision"))
	require.Error(t, err)
	assert.Contains(t, stderr, "declared both as a flat entry and nested")
}

func TestCmd_Validate_InheritOptOutFailsTagRule(t *testing.T) {
	_, stderr, err := runValidate(t, filepath.Join("testdata", "inherit_opt_out"))
	require.Error(t, err)
	assert.Contains(t, stderr, "requires tag")
}

func TestCmd_Schema_ProducesValidJSON(t *testing.T) {
	cmd := cmdUcm.New()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"schema"})

	require.NoError(t, cmd.Execute())

	// The output must parse as JSON and declare an object at the root.
	var schema map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &schema))

	// Should at minimum advertise the ucm root in the $defs tree.
	raw := out.String()
	assert.True(t, strings.Contains(raw, "resources.Catalog"), "schema should describe Catalog")
	assert.True(t, strings.Contains(raw, "resources.TagValidationRule"), "schema should describe TagValidationRule")
}
