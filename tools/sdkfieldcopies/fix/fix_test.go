package fix_test

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/tools/sdkfieldcopies/fix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureFile = "bundle/direct/dresources/resource.go"

func formatFixture(t *testing.T, source string) string {
	t.Helper()
	formatted, err := format.Source([]byte(source))
	require.NoError(t, err)
	return string(formatted)
}

func writeFixture(t *testing.T, source string) string {
	t.Helper()
	root := t.TempDir()
	filename := filepath.Join(root, filepath.FromSlash(fixtureFile))
	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/sdk-copy-fixture\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filename, []byte(formatFixture(t, source)), 0o644))
	return root
}

func readFixture(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fixtureFile)))
	require.NoError(t, err)
	return string(data)
}

func TestRunWarehouseFieldsAndIdempotency(t *testing.T) {
	const source = `package dresources

type Warehouse struct {
	Name string "json:\"name\""
	ReydenClusterSizeMultiplier float64 "json:\"reyden_cluster_size_multiplier,omitempty\""
	MaxDbu int "json:\"max_dbu,omitempty\""
	CustomerVersionLabel string "json:\"customer_version_label,omitempty\""
	ForceSendFields []string "json:\"-\""
}
type EditWarehouse Warehouse

func remap(remote *Warehouse) *Warehouse {
	return &Warehouse{ // sdk:copy-from remote
		// Preserve remote name and the explicit transport policy.
		Name: remote.Name,
		ForceSendFields: nil,
	}
}

func update(config *Warehouse) EditWarehouse {
	return EditWarehouse{ // sdk:copy-from config
		Name: config.Name,
		ForceSendFields: config.ForceSendFields,
	}
}
`
	root := writeFixture(t, source)
	var output bytes.Buffer
	require.NoError(t, fix.Run(t.Context(), root, &output))

	expected := source
	for _, name := range []string{"remote", "config"} {
		marker := "// sdk:copy-from " + name + "\n"
		expected = strings.Replace(expected, marker, marker+
			"CustomerVersionLabel: "+name+".CustomerVersionLabel,\n"+
			"MaxDbu: "+name+".MaxDbu,\n"+
			"ReydenClusterSizeMultiplier: "+name+".ReydenClusterSizeMultiplier,\n", 1)
	}
	assert.Equal(t, formatFixture(t, expected), readFixture(t, root))

	first := readFixture(t, root)
	output.Reset()
	require.NoError(t, fix.Run(t.Context(), root, &output))
	assert.Equal(t, first, readFixture(t, root))
}

func TestRunPreservesExplicitMappingsAndUnmarkedLiterals(t *testing.T) {
	//nolint:dupword // Go fixture fields intentionally share names with their types.
	const source = `package dresources

type Kind string
type Options struct {
	Name string "json:\"name\""
	Added string "json:\"added\""
}
type Input struct {
	Name string "json:\"name\""
	Added string "json:\"added\""
	Disabled bool "json:\"disabled\""
	Count int "json:\"count\""
	Pointer *Options "json:\"pointer\""
	Options Options "json:\"options\""
	Kind string "json:\"kind\""
	ForceSendFields []string "json:\"-\""
}
type Request struct {
	Name string "json:\"name\""
	Added string "json:\"added\""
	Disabled bool "json:\"disabled\""
	Count int "json:\"count\""
	Pointer *Options "json:\"pointer\""
	Options Options "json:\"options\""
	Kind Kind "json:\"kind\""
	ForceSendFields []string "json:\"-\""
}
func filter(fields []string) []string { return fields }

func mapped(config, remote *Input) Request {
	return Request{ // sdk:copy-from config
		Name: remote.Name, // This existing field has a different source.
		Disabled: false,
		Count: 0,
		Pointer: nil,
		Options: Options{Name: "deliberate"}, // Nested literal has no directive.
		Kind: Kind(remote.Kind),
		ForceSendFields: filter(remote.ForceSendFields),
	}
}

func selective(config *Input) Request {
	return Request{
		Name: config.Name,
	}
}
`
	root := writeFixture(t, source)
	var output bytes.Buffer
	require.NoError(t, fix.Run(t.Context(), root, &output))
	expected := strings.Replace(source, "// sdk:copy-from config\n", "// sdk:copy-from config\nAdded: config.Added,\n", 1)
	assert.Equal(t, formatFixture(t, expected), readFixture(t, root))
}

func TestRunRequiresMatchingNamesTypesAndWireFields(t *testing.T) {
	//nolint:dupword // Go fixture fields intentionally share names with their types.
	const source = `package dresources

type First string
type Second string
type Alias = string
type Source struct {
	Name string "json:\"name\""
	Eligible string "json:\"eligible\""
	Alias string "json:\"alias\""
	Mismatch string "json:\"mismatch\""
	Assignable string "json:\"assignable\""
	Named First "json:\"named\""
	WireMismatch string "json:\"source_name\""
	NoTag string
	EmptyTag string "json:\",omitempty\""
	Ignored string "json:\"-\""
	ForceSendFields []string "json:\"force_send_fields\""
	UpdateMask string "json:\"update_mask\""
	hidden string "json:\"hidden\""
}
type Destination struct {
	Name string "json:\"name\""
	Eligible string "json:\"eligible,omitempty\""
	Alias Alias "json:\"alias\""
	Mismatch int "json:\"mismatch\""
	Assignable any "json:\"assignable\""
	Named Second "json:\"named\""
	WireMismatch string "json:\"destination_name\""
	NoTag string
	EmptyTag string "json:\",omitempty\""
	Ignored string "json:\"-\""
	Missing string "json:\"missing\""
	ForceSendFields []string "json:\"force_send_fields\""
	UpdateMask string "json:\"update_mask\""
	hidden string "json:\"hidden\""
}

func remap(src *Source) Destination {
	return Destination{ // sdk:copy-from src
		Name: src.Name,
	}
}
`
	root := writeFixture(t, source)
	var output bytes.Buffer
	require.NoError(t, fix.Run(t.Context(), root, &output))
	expected := strings.Replace(source, "// sdk:copy-from src\n", "// sdk:copy-from src\nAlias: src.Alias,\nEligible: src.Eligible,\n", 1)
	assert.Equal(t, formatFixture(t, expected), readFixture(t, root))
	for _, field := range []string{"Mismatch", "Assignable", "Named", "WireMismatch", "NoTag", "EmptyTag", "Missing"} {
		assert.Contains(t, output.String(), field)
	}
}

func TestRunPromotedFieldsAndWholeEmbeddings(t *testing.T) {
	const source = `package dresources

type Common struct {
	Name string "json:\"name\""
	Added string "json:\"added\""
}
type Source struct { Common }
type Destination struct {
	Common
	Id string "json:\"id\""
}

func partial(src *Source) Destination {
	return Destination{ // sdk:copy-from src
		Name: src.Name,
		Id: "",
	}
}

func whole(src *Source) Destination {
	return Destination{ // sdk:copy-from src
		Common: src.Common,
		Id: "",
	}
}
`
	root := writeFixture(t, source)
	var output bytes.Buffer
	require.NoError(t, fix.Run(t.Context(), root, &output))
	expected := strings.Replace(source, "// sdk:copy-from src\n", "// sdk:copy-from src\nAdded: src.Added,\n", 1)
	assert.Equal(t, formatFixture(t, expected), readFixture(t, root))
}

func TestRunRejectsUnsafeSourcePromotions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
	}{
		{
			name: "shadowed field",
			source: `type Source struct {
				Common
				Name string "json:\"name\""
				Added int "json:\"added\""
			}`,
		},
		{
			name: "shadowed by method",
			source: `type Source struct {
				Common
				Name string "json:\"name\""
			}
			func (Source) Added() string { return "" }`,
		},
		{
			name: "ambiguous field",
			source: `type Other Common
			type Source struct {
				Common
				Other
				Name string "json:\"name\""
			}`,
		},
		{
			name: "embedded pointer",
			source: `type Source struct {
				*Common
				Name string "json:\"name\""
			}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `package dresources

type Common struct { Added string "json:\"added\"" }
type Destination struct {
	Name string "json:\"name\""
	Added string "json:\"added\""
}
` + tc.source + `

func remap(src *Source) Destination {
	return Destination{ // sdk:copy-from src
		Name: src.Name,
	}
}
`
			root := writeFixture(t, source)
			before := readFixture(t, root)
			var output bytes.Buffer
			require.NoError(t, fix.Run(t.Context(), root, &output))
			assert.Equal(t, before, readFixture(t, root))
			assert.Contains(t, output.String(), "Added")
		})
	}
}

func TestRunGuardedSelectorSource(t *testing.T) {
	const source = `package dresources

type Spec struct {
	Name string "json:\"name\""
	Added string "json:\"added\""
}
type Response struct { Spec *Spec }
type Request Spec

func remap(remote *Response) *Request {
	if remote.Spec == nil {
		return nil
	}
	return &Request{ // sdk:copy-from remote.Spec
		Name: remote.Spec.Name,
	}
}
`
	root := writeFixture(t, source)
	var output bytes.Buffer
	require.NoError(t, fix.Run(t.Context(), root, &output))
	expected := strings.Replace(source, "// sdk:copy-from remote.Spec\n", "// sdk:copy-from remote.Spec\nAdded: remote.Spec.Added,\n", 1)
	assert.Equal(t, formatFixture(t, expected), readFixture(t, root))
}

func TestRunErrorsLeaveAllFilesUnchanged(t *testing.T) {
	const source = `package dresources

type Request struct {
	Name string "json:\"name\""
	Added string "json:\"added\""
}

func remap(src *Request) Request {
	return Request{ // sdk:copy-from src
		Name: src.Name,
	}
}
`
	for _, tc := range []struct {
		name    string
		invalid string
	}{
		{
			name: "missing source",
			invalid: `func broken(src *Request) Request {
				return Request{ // sdk:copy-from
					Name: src.Name,
				}
			}`,
		},
		{
			name: "source call",
			invalid: `func broken(src func() *Request) Request {
				return Request{ // sdk:copy-from src()
					Name: "",
				}
			}`,
		},
		{
			name: "dangling directive",
			invalid: `// sdk:copy-from src
			var unrelated = 1`,
		},
		{
			name:    "package error",
			invalid: `var broken = undefined`,
		},
		{
			name: "proposed edit introduces initialization cycle",
			invalid: `var src = Request{ // sdk:copy-from src
				Name: "",
			}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeFixture(t, source)
			before := readFixture(t, root)
			invalid := formatFixture(t, "package dresources\n\n"+tc.invalid)
			filename := filepath.Join(root, "bundle", "direct", "dresources", "invalid.go")
			require.NoError(t, os.WriteFile(filename, []byte(invalid), 0o644))
			var output bytes.Buffer
			require.Error(t, fix.Run(t.Context(), root, &output))
			assert.Equal(t, before, readFixture(t, root))
			after, err := os.ReadFile(filename)
			require.NoError(t, err)
			assert.Equal(t, invalid, string(after))
		})
	}
}

func TestRunBindsDirectiveToInnerLiteralOnSameLine(t *testing.T) {
	//nolint:dupword // Go fixture fields intentionally share names with their types.
	const source = `package dresources

type Inner struct {
	Name string "json:\"name\""
	Added string "json:\"inner_added\""
}
type Outer struct {
	Inner Inner "json:\"inner\""
	Added string "json:\"added\""
}

func remap(src *Inner) Outer {
	return Outer{Inner: Inner{ // sdk:copy-from src
		Name: src.Name,
	}}
}
`
	root := writeFixture(t, source)
	var output bytes.Buffer
	require.NoError(t, fix.Run(t.Context(), root, &output))
	expected := strings.Replace(source, "// sdk:copy-from src\n", "// sdk:copy-from src\nAdded: src.Added,\n", 1)
	assert.Equal(t, formatFixture(t, expected), readFixture(t, root))
}

func TestRunUsesPhysicalPositionsWithLineDirective(t *testing.T) {
	const source = `package dresources

type Request struct {
	Name string "json:\"name\""
	Added string "json:\"added\""
}

//line elsewhere.go:900
func remap(src *Request) Request {
	return Request{ // sdk:copy-from src
		Name: src.Name,
	}
}
`
	root := writeFixture(t, source)
	var output bytes.Buffer
	require.NoError(t, fix.Run(t.Context(), root, &output))
	expected := strings.Replace(source, "// sdk:copy-from src\n", "// sdk:copy-from src\nAdded: src.Added,\n", 1)
	assert.Equal(t, formatFixture(t, expected), readFixture(t, root))
	assert.Contains(t, output.String(), "resource.go:")
	assert.NotContains(t, output.String(), "elsewhere.go:")
}

func TestRunLeavesDestinationMethodCollisionForReview(t *testing.T) {
	const source = `package dresources

type Fields struct {
	Name string "json:\"name\""
	Added string "json:\"added\""
}
type Destination struct { Fields }
func (Destination) Added() string { return "" }

func remap(src *Fields) Destination {
	return Destination{ // sdk:copy-from src
		Name: src.Name,
	}
}
`
	root := writeFixture(t, source)
	before := readFixture(t, root)
	var output bytes.Buffer
	require.NoError(t, fix.Run(t.Context(), root, &output))
	assert.Equal(t, before, readFixture(t, root))
}
