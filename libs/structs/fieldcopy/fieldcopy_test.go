package fieldcopy_test

import (
	"testing"

	"github.com/databricks/cli/libs/structs/fieldcopy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type srcBasic struct {
	Name    string
	Age     int
	Email   string
	private string //nolint:unused
}

type dstBasic struct {
	Name  string
	Age   int
	Email string
}

func TestDoBasicCopy(t *testing.T) {
	c := fieldcopy.Copy[srcBasic, dstBasic]{}
	src := srcBasic{Name: "alice", Age: 30, Email: "a@b.c"}
	dst := c.Do(&src)
	assert.Equal(t, "alice", dst.Name)
	assert.Equal(t, 30, dst.Age)
	assert.Equal(t, "a@b.c", dst.Email)
}

func TestDoCachedSecondCall(t *testing.T) {
	c := fieldcopy.Copy[srcBasic, dstBasic]{}
	src1 := srcBasic{Name: "alice", Age: 30, Email: "a@b.c"}
	dst1 := c.Do(&src1)
	assert.Equal(t, "alice", dst1.Name)

	src2 := srcBasic{Name: "bob", Age: 25, Email: "b@c.d"}
	dst2 := c.Do(&src2)
	assert.Equal(t, "bob", dst2.Name)
	assert.Equal(t, 25, dst2.Age)
}

type srcWithExtra struct {
	Name  string
	Age   int
	Extra string
}

type dstSmall struct {
	Name string
	Age  int
}

func TestDoSkipSrc(t *testing.T) {
	c := fieldcopy.Copy[srcWithExtra, dstSmall]{
		SkipSrc: []string{"Extra"},
	}
	src := srcWithExtra{Name: "alice", Age: 30, Extra: "ignored"}
	dst := c.Do(&src)
	assert.Equal(t, "alice", dst.Name)
	assert.Equal(t, 30, dst.Age)
}

type dstWithDefault struct {
	Name    string
	Age     int
	Default string
}

func TestDoSkipDst(t *testing.T) {
	c := fieldcopy.Copy[srcBasic, dstWithDefault]{
		SkipSrc: []string{"Email"},
		SkipDst: []string{"Default"},
	}
	src := srcBasic{Name: "alice", Age: 30, Email: "a@b.c"}
	dst := c.Do(&src)
	assert.Equal(t, "alice", dst.Name)
	assert.Equal(t, 30, dst.Age)
	assert.Equal(t, "", dst.Default)
}

type srcRenamed struct {
	FullName string
	Age      int
}

type dstRenamed struct {
	Name string
	Age  int
}

func TestDoRename(t *testing.T) {
	c := fieldcopy.Copy[srcRenamed, dstRenamed]{
		Rename: map[string]string{"Name": "FullName"},
	}
	src := srcRenamed{FullName: "alice", Age: 30}
	dst := c.Do(&src)
	assert.Equal(t, "alice", dst.Name)
	assert.Equal(t, 30, dst.Age)
}

func TestValidateBasicPass(t *testing.T) {
	c := fieldcopy.Copy[srcBasic, dstBasic]{}
	require.NoError(t, c.Validate())
}

func TestValidateWithSkips(t *testing.T) {
	c := fieldcopy.Copy[srcWithExtra, dstWithDefault]{
		SkipSrc: []string{"Extra"},
		SkipDst: []string{"Default"},
	}
	require.NoError(t, c.Validate())
}

func TestValidateWithRename(t *testing.T) {
	c := fieldcopy.Copy[srcRenamed, dstRenamed]{
		Rename: map[string]string{"Name": "FullName"},
	}
	require.NoError(t, c.Validate())
}

func TestValidateUnhandledDstField(t *testing.T) {
	c := fieldcopy.Copy[dstSmall, dstWithDefault]{}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `dst field "Default"`)
	assert.Contains(t, err.Error(), "SkipDst")
}

func TestValidateUnconsumedSrcField(t *testing.T) {
	c := fieldcopy.Copy[srcWithExtra, dstSmall]{}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `src field "Extra"`)
	assert.Contains(t, err.Error(), "SkipSrc")
}

func TestValidateStaleSkipSrc(t *testing.T) {
	c := fieldcopy.Copy[srcBasic, dstBasic]{
		SkipSrc: []string{"NonExistent"},
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `stale SkipSrc entry "NonExistent"`)
}

func TestValidateStaleSkipDst(t *testing.T) {
	c := fieldcopy.Copy[srcBasic, dstBasic]{
		SkipDst: []string{"NonExistent"},
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `stale SkipDst entry "NonExistent"`)
}

func TestValidateStaleRenameKey(t *testing.T) {
	c := fieldcopy.Copy[srcRenamed, dstRenamed]{
		Rename: map[string]string{
			"Name":        "FullName",
			"NonExistent": "Age",
		},
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `stale Rename key "NonExistent"`)
}

func TestValidateStaleRenameValue(t *testing.T) {
	c := fieldcopy.Copy[srcRenamed, dstRenamed]{
		Rename: map[string]string{
			"Name": "NonExistent",
		},
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `stale Rename value "NonExistent"`)
}

type srcTypeMismatch struct {
	Name string
	Age  string // string instead of int
}

func TestValidateTypeMismatch(t *testing.T) {
	c := fieldcopy.Copy[srcTypeMismatch, dstBasic]{
		SkipDst: []string{"Age", "Email"},
	}
	require.NoError(t, c.Validate())

	// Without SkipDst, the type mismatch is caught.
	c2 := fieldcopy.Copy[srcTypeMismatch, dstBasic]{
		SkipDst: []string{"Email"},
	}
	err := c2.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `dst field "Age"`)
	assert.Contains(t, err.Error(), "not assignable")
}

type srcPointer struct {
	Name  string
	Items *[]string
}

type dstPointer struct {
	Name  string
	Items *[]string
}

func TestDoPointerFields(t *testing.T) {
	c := fieldcopy.Copy[srcPointer, dstPointer]{}
	items := []string{"a", "b"}
	src := srcPointer{Name: "alice", Items: &items}
	dst := c.Do(&src)
	assert.Equal(t, "alice", dst.Name)
	require.NotNil(t, dst.Items)
	assert.Equal(t, []string{"a", "b"}, *dst.Items)
	// Pointer is shared (shallow copy).
	assert.Same(t, src.Items, dst.Items)
}

func TestDoNilPointerFields(t *testing.T) {
	c := fieldcopy.Copy[srcPointer, dstPointer]{}
	src := srcPointer{Name: "alice", Items: nil}
	dst := c.Do(&src)
	assert.Equal(t, "alice", dst.Name)
	assert.Nil(t, dst.Items)
}

type srcMap struct {
	Tags map[string]string
}

type dstMap struct {
	Tags map[string]string
}

func TestDoMapFields(t *testing.T) {
	c := fieldcopy.Copy[srcMap, dstMap]{}
	src := srcMap{Tags: map[string]string{"k": "v"}}
	dst := c.Do(&src)
	assert.Equal(t, map[string]string{"k": "v"}, dst.Tags)
	// Map is shared (shallow copy).
	src.Tags["k2"] = "v2"
	assert.Equal(t, "v2", dst.Tags["k2"])
}

type srcSlice struct {
	Items []string
}

type dstSlice struct {
	Items []string
}

func TestDoSliceFields(t *testing.T) {
	c := fieldcopy.Copy[srcSlice, dstSlice]{}
	src := srcSlice{Items: []string{"a", "b"}}
	dst := c.Do(&src)
	assert.Equal(t, []string{"a", "b"}, dst.Items)
}

func TestDoZeroValue(t *testing.T) {
	c := fieldcopy.Copy[srcBasic, dstBasic]{}
	src := srcBasic{}
	dst := c.Do(&src)
	assert.Equal(t, "", dst.Name)
	assert.Equal(t, 0, dst.Age)
	assert.Equal(t, "", dst.Email)
}

type srcBool struct {
	Enabled bool
	Name    string
}

type dstBool struct {
	Enabled bool
	Name    string
}

func TestDoBoolZeroValue(t *testing.T) {
	c := fieldcopy.Copy[srcBool, dstBool]{}
	src := srcBool{Enabled: false, Name: "test"}
	dst := c.Do(&src)
	assert.Equal(t, false, dst.Enabled)
	assert.Equal(t, "test", dst.Name)
}

type srcNested struct {
	Name   string
	Config *nestedConfig
}

type dstNested struct {
	Name   string
	Config *nestedConfig
}

type nestedConfig struct {
	Value int
}

func TestDoNestedStructPointer(t *testing.T) {
	c := fieldcopy.Copy[srcNested, dstNested]{}
	src := srcNested{Name: "alice", Config: &nestedConfig{Value: 42}}
	dst := c.Do(&src)
	assert.Equal(t, "alice", dst.Name)
	require.NotNil(t, dst.Config)
	assert.Equal(t, 42, dst.Config.Value)
}

func TestValidateMultipleErrors(t *testing.T) {
	c := fieldcopy.Copy[srcWithExtra, dstWithDefault]{
		// Missing: SkipSrc for Extra, SkipDst for Default
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"Extra"`)
	assert.Contains(t, err.Error(), `"Default"`)
}

// Verify that private (unexported) fields are ignored.
type srcPrivate struct {
	Name    string
	private int //nolint:unused
}

type dstPrivate struct {
	Name    string
	private int //nolint:unused
}

func TestValidateIgnoresUnexportedFields(t *testing.T) {
	c := fieldcopy.Copy[srcPrivate, dstPrivate]{}
	require.NoError(t, c.Validate())

	dst := c.Do(&srcPrivate{Name: "test"})
	assert.Equal(t, "test", dst.Name)
}

// ForceSendFields auto-handling tests.

type srcFSF struct {
	Name            string
	Age             int
	Extra           string
	ForceSendFields []string
}

type dstFSF struct {
	Name            string
	Age             int
	ForceSendFields []string
}

func TestDoForceSendFieldsAutoFiltered(t *testing.T) {
	c := fieldcopy.Copy[srcFSF, dstFSF]{
		SkipSrc: []string{"Extra"},
	}
	src := srcFSF{
		Name:            "alice",
		Age:             30,
		ForceSendFields: []string{"Name", "Age", "Extra"},
	}
	dst := c.Do(&src)
	assert.Equal(t, "alice", dst.Name)
	assert.Equal(t, 30, dst.Age)
	// "Extra" should be filtered out since it doesn't exist on dstFSF.
	assert.Equal(t, []string{"Name", "Age"}, dst.ForceSendFields)
}

func TestDoForceSendFieldsNil(t *testing.T) {
	c := fieldcopy.Copy[srcFSF, dstFSF]{
		SkipSrc: []string{"Extra"},
	}
	src := srcFSF{Name: "alice", ForceSendFields: nil}
	dst := c.Do(&src)
	assert.Equal(t, "alice", dst.Name)
	assert.Nil(t, dst.ForceSendFields)
}

func TestDoForceSendFieldsEmpty(t *testing.T) {
	c := fieldcopy.Copy[srcFSF, dstFSF]{
		SkipSrc: []string{"Extra"},
	}
	src := srcFSF{Name: "alice", ForceSendFields: []string{}}
	dst := c.Do(&src)
	assert.Nil(t, dst.ForceSendFields)
}

func TestDoForceSendFieldsExcludesSkipDst(t *testing.T) {
	c := fieldcopy.Copy[srcFSF, dstFSF]{
		SkipSrc: []string{"Extra"},
		SkipDst: []string{"Age"},
	}
	src := srcFSF{
		Name:            "alice",
		Age:             30,
		ForceSendFields: []string{"Name", "Age"},
	}
	dst := c.Do(&src)
	// "Age" should be filtered out since it's in SkipDst.
	assert.Equal(t, []string{"Name"}, dst.ForceSendFields)
	// Age should NOT be copied (it's in SkipDst).
	assert.Equal(t, 0, dst.Age)
}

func TestDoForceSendFieldsAllFiltered(t *testing.T) {
	c := fieldcopy.Copy[srcFSF, dstFSF]{
		SkipSrc: []string{"Extra"},
	}
	src := srcFSF{ForceSendFields: []string{"Extra", "NonExistent"}}
	dst := c.Do(&src)
	// All entries filtered out → nil.
	assert.Nil(t, dst.ForceSendFields)
}

func TestValidateForceSendFieldsAutoHandled(t *testing.T) {
	c := fieldcopy.Copy[srcFSF, dstFSF]{
		SkipSrc: []string{"Extra"},
	}
	// ForceSendFields should not need to be in SkipSrc or SkipDst.
	require.NoError(t, c.Validate())
}

func TestDoForceSendFieldsManualWhenInSkipDst(t *testing.T) {
	c := fieldcopy.Copy[srcFSF, dstFSF]{
		SkipSrc: []string{"Extra", "ForceSendFields"},
		SkipDst: []string{"ForceSendFields"},
	}
	src := srcFSF{
		Name:            "alice",
		ForceSendFields: []string{"Name", "Extra"},
	}
	dst := c.Do(&src)
	// ForceSendFields in SkipDst → not auto-handled, left at nil.
	assert.Nil(t, dst.ForceSendFields)
	require.NoError(t, c.Validate())
}
