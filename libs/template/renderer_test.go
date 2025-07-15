package template

import (
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"

	"github.com/databricks/cli/bundle"
	bundleConfig "github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/tags"
	"github.com/databricks/databricks-sdk-go"
	workspaceConfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	defaultFilePermissions fs.FileMode
	defaultDirPermissions  fs.FileMode
)

func init() {
	if runtime.GOOS == "windows" {
		defaultFilePermissions = fs.FileMode(0o666)
		defaultDirPermissions = fs.FileMode(0o777)
	} else {
		defaultFilePermissions = fs.FileMode(0o644)
		defaultDirPermissions = fs.FileMode(0o755)
	}
}

func assertBuiltinTemplateValid(t *testing.T, template string, settings map[string]any, target string, isServicePrincipal, build bool, tempDir string) {
	ctx := dbr.MockRuntime(context.Background(), dbr.Environment{})

	templateFS, err := fs.Sub(builtinTemplates, path.Join("templates", template))
	require.NoError(t, err)

	w := &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{Host: "https://myhost.com"},
	}

	// Prepare helpers
	cachedUser = &iam.User{UserName: "user@domain.com"}
	if isServicePrincipal {
		cachedUser.UserName = "1d410060-a513-496f-a197-23cc82e5f46d"
	}
	cachedIsServicePrincipal = &isServicePrincipal
	ctx = cmdctx.SetWorkspaceClient(ctx, w)
	helpers := loadHelpers(ctx)

	renderer, err := newRenderer(ctx, settings, helpers, templateFS, templateDirName, libraryDirName)
	require.NoError(t, err)

	// Evaluate template
	err = renderer.walk()
	require.NoError(t, err)
	out, err := filer.NewLocalClient(tempDir)
	require.NoError(t, err)
	err = renderer.persistToDisk(ctx, out)
	require.NoError(t, err)

	// Verify permissions on file and directory
	testutil.AssertFilePermissions(t, filepath.Join(tempDir, "my_project/README.md"), defaultFilePermissions)
	testutil.AssertDirPermissions(t, filepath.Join(tempDir, "my_project/resources"), defaultDirPermissions)

	b, err := bundle.Load(ctx, filepath.Join(tempDir, "my_project"))
	require.NoError(t, err)

	// Initialize logdiag context for phase functions
	ctx = logdiag.InitContext(ctx)
	logdiag.SetCollect(ctx, true)

	phases.LoadNamedTarget(ctx, b, target)
	diags := logdiag.FlushCollected(ctx)
	require.Empty(t, diags)

	// Apply initialize / validation mutators
	bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
		b.Config.Workspace.CurrentUser = &bundleConfig.User{User: cachedUser}
		b.Config.Bundle.Terraform = &bundleConfig.Terraform{
			ExecPath: "sh",
		}
	})

	b.Tagging = tags.ForCloud(w.Config)
	b.SetWorkpaceClient(w)
	b.WorkspaceClient()

	phases.Initialize(ctx, b)
	diags = logdiag.FlushCollected(ctx)
	require.Empty(t, diags)

	// Apply build mutator
	if build {
		phases.Build(ctx, b)
		diags = logdiag.FlushCollected(ctx)
		require.Empty(t, diags)
	}
}

func TestBuiltinSQLTemplateValid(t *testing.T) {
	for _, personal_schemas := range []string{"yes", "no"} {
		for _, target := range []string{"dev", "prod"} {
			for _, isServicePrincipal := range []bool{true, false} {
				config := map[string]any{
					"project_name":     "my_project",
					"http_path":        "/sql/1.0/warehouses/123abc",
					"default_catalog":  "users",
					"shared_schema":    "lennart",
					"personal_schemas": personal_schemas,
				}
				build := false
				assertBuiltinTemplateValid(t, "default-sql", config, target, isServicePrincipal, build, t.TempDir())
			}
		}
	}
}

func TestBuiltinDbtTemplateValid(t *testing.T) {
	catalog := "hive_metastore"
	cachedCatalog = &catalog
	for _, personal_schemas := range []string{"yes", "no"} {
		for _, target := range []string{"dev", "prod"} {
			for _, isServicePrincipal := range []bool{true, false} {
				config := map[string]any{
					"project_name":     "my_project",
					"http_path":        "/sql/1.0/warehouses/123",
					"default_catalog":  "hive_metastore",
					"personal_schemas": personal_schemas,
					"shared_schema":    "lennart",
				}
				build := false
				assertBuiltinTemplateValid(t, "dbt-sql", config, target, isServicePrincipal, build, t.TempDir())
			}
		}
	}
}

func TestRendererWithAssociatedTemplateInLibrary(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/email/template", "./testdata/email/library")
	require.NoError(t, err)

	err = r.walk()
	require.NoError(t, err)
	out, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	err = r.persistToDisk(ctx, out)
	require.NoError(t, err)

	b, err := os.ReadFile(filepath.Join(tmpDir, "my_email"))
	require.NoError(t, err)
	assert.Equal(t, "shreyas.goenka@databricks.com", strings.Trim(string(b), "\n\r"))
}

func TestRendererExecuteTemplate(t *testing.T) {
	templateText := `"{{.count}} items are made of {{.Material}}".
{{if eq .Animal "sheep" }}
Sheep wool is the best!
{{else}}
{{.Animal}} wool is not too bad...
{{end}}
My email is {{template "email"}}
`

	r := renderer{
		config: map[string]any{
			"Material": "wool",
			"count":    1,
			"Animal":   "sheep",
		},
		baseTemplate: template.Must(template.New("base").Parse(`{{define "email"}}shreyas.goenka@databricks.com{{end}}`)),
	}

	statement, err := r.executeTemplate(templateText)
	require.NoError(t, err)
	assert.Contains(t, statement, `"1 items are made of wool"`)
	assert.NotContains(t, statement, `cat wool is not too bad.."`)
	assert.Contains(t, statement, "Sheep wool is the best!")
	assert.Contains(t, statement, `My email is shreyas.goenka@databricks.com`)

	r = renderer{
		config: map[string]any{
			"Material": "wool",
			"count":    1,
			"Animal":   "cat",
		},
		baseTemplate: template.Must(template.New("base").Parse(`{{define "email"}}hrithik.roshan@databricks.com{{end}}`)),
	}

	statement, err = r.executeTemplate(templateText)
	require.NoError(t, err)
	assert.Contains(t, statement, `"1 items are made of wool"`)
	assert.Contains(t, statement, `cat wool is not too bad...`)
	assert.NotContains(t, statement, "Sheep wool is the best!")
	assert.Contains(t, statement, `My email is hrithik.roshan@databricks.com`)
}

func TestRendererExecuteTemplateWithUnknownProperty(t *testing.T) {
	templateText := `{{.does_not_exist}}`

	r := renderer{
		config:       map[string]any{},
		baseTemplate: template.New("base"),
	}

	_, err := r.executeTemplate(templateText)
	assert.ErrorContains(t, err, "variable \"does_not_exist\" not defined")
}

func TestRendererIsSkipped(t *testing.T) {
	skipPatterns := []string{"a*", "*yz", "def", "a/b/*"}

	// skipped paths
	match, err := isSkipped("abc", skipPatterns)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = isSkipped("abcd", skipPatterns)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = isSkipped("a", skipPatterns)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = isSkipped("xxyz", skipPatterns)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = isSkipped("yz", skipPatterns)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = isSkipped("a/b/c", skipPatterns)
	require.NoError(t, err)
	assert.True(t, match)

	// NOT skipped paths
	match, err = isSkipped(".", skipPatterns)
	require.NoError(t, err)
	assert.False(t, match)

	match, err = isSkipped("y", skipPatterns)
	require.NoError(t, err)
	assert.False(t, match)

	match, err = isSkipped("z", skipPatterns)
	require.NoError(t, err)
	assert.False(t, match)

	match, err = isSkipped("defg", skipPatterns)
	require.NoError(t, err)
	assert.False(t, match)

	match, err = isSkipped("cat", skipPatterns)
	require.NoError(t, err)
	assert.False(t, match)

	match, err = isSkipped("a/b/c/d", skipPatterns)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestRendererPersistToDisk(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	r := &renderer{
		ctx:          ctx,
		skipPatterns: []string{"a/b/c", "mn*"},
		files: []file{
			&inMemoryFile{
				perm:    0o444,
				relPath: "a/b/c",
				content: nil,
			},
			&inMemoryFile{
				perm:    0o444,
				relPath: "mno",
				content: nil,
			},
			&inMemoryFile{
				perm:    0o444,
				relPath: "a/b/d",
				content: []byte("123"),
			},
			&inMemoryFile{
				perm:    0o444,
				relPath: "mmnn",
				content: []byte("456"),
			},
		},
	}

	out, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	err = r.persistToDisk(ctx, out)
	require.NoError(t, err)

	assert.NoFileExists(t, filepath.Join(tmpDir, "a", "b", "c"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "mno"))

	testutil.AssertFileContents(t, filepath.Join(tmpDir, "a/b/d"), "123")
	testutil.AssertFilePermissions(t, filepath.Join(tmpDir, "a/b/d"), fs.FileMode(0o444))
	testutil.AssertFileContents(t, filepath.Join(tmpDir, "mmnn"), "456")
	testutil.AssertFilePermissions(t, filepath.Join(tmpDir, "mmnn"), fs.FileMode(0o444))
}

func TestRendererWalk(t *testing.T) {
	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, nil)

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/walk/template", "./testdata/walk/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	getContent := func(r *renderer, path string) string {
		for _, f := range r.files {
			if f.RelPath() != path {
				continue
			}
			b, err := f.contents()
			require.NoError(t, err)
			return strings.Trim(string(b), "\r\n")
		}
		require.FailNow(t, "file is absent: "+path)
		return ""
	}

	assert.Len(t, r.files, 4)
	assert.Equal(t, "file one", getContent(r, "file1"))
	assert.Equal(t, "file two", getContent(r, "file2"))
	assert.Equal(t, "file three", getContent(r, "dir1/dir3/file3"))
	assert.Equal(t, "file four", getContent(r, "dir2/file4"))
}

func TestRendererFailFunction(t *testing.T) {
	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, nil)

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/fail/template", "./testdata/fail/library")
	require.NoError(t, err)

	err = r.walk()
	assert.Equal(t, "I am an error message", err.Error())
}

func TestRendererSkipsDirsEagerly(t *testing.T) {
	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, nil)

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/skip-dir-eagerly/template", "./testdata/skip-dir-eagerly/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	content := string(r.files[0].(*inMemoryFile).content)
	assert.Equal(t, "I should be the only file created", strings.Trim(content, "\r\n"))
}

func TestRendererSkipAllFilesInCurrentDirectory(t *testing.T) {
	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	tmpDir := t.TempDir()

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/skip-all-files-in-cwd/template", "./testdata/skip-all-files-in-cwd/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)
	// All 3 files are executed and have in memory representations
	require.Len(t, r.files, 3)

	out, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	err = r.persistToDisk(ctx, out)
	require.NoError(t, err)

	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	// Assert none of the files are persisted to disk, because of {{skip "*"}}
	assert.Empty(t, entries)
}

func TestRendererSkipPatternsAreRelativeToFileDirectory(t *testing.T) {
	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, nil)

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/skip-is-relative/template", "./testdata/skip-is-relative/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.skipPatterns, 3)
	assert.Contains(t, r.skipPatterns, "a")
	assert.Contains(t, r.skipPatterns, "dir1/b")
	assert.Contains(t, r.skipPatterns, "dir1/dir2/c")
}

func TestRendererSkip(t *testing.T) {
	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	tmpDir := t.TempDir()

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/skip/template", "./testdata/skip/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)
	// All 6 files are computed, even though "dir2/*" is present as a skip pattern
	// This is because "dir2/*" matches the files in dir2, but not dir2 itself
	assert.Len(t, r.files, 6)

	out, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	err = r.persistToDisk(ctx, out)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tmpDir, "file1"))
	assert.FileExists(t, filepath.Join(tmpDir, "file2"))
	assert.FileExists(t, filepath.Join(tmpDir, "dir1/file5"))

	// These files have been skipped
	assert.NoFileExists(t, filepath.Join(tmpDir, "file3"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "dir1/file4"))
	assert.NoDirExists(t, filepath.Join(tmpDir, "dir2"))
	assert.NoFileExists(t, filepath.Join(tmpDir, "dir2/file6"))
}

func TestRendererReadsPermissionsBits(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.SkipNow()
	}
	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, nil)

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/executable-bit-read/template", "./testdata/executable-bit-read/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	getPermissions := func(r *renderer, path string) fs.FileMode {
		for _, f := range r.files {
			if f.RelPath() != path {
				continue
			}
			switch v := f.(type) {
			case *inMemoryFile:
				return v.perm
			case *copyFile:
				return v.perm
			default:
				require.FailNow(t, "execution should not reach here")
			}
		}
		require.FailNow(t, "file is absent: "+path)
		return 0
	}

	assert.Len(t, r.files, 2)
	assert.Equal(t, getPermissions(r, "script.sh"), fs.FileMode(0o755))
	assert.Equal(t, getPermissions(r, "not-a-script"), fs.FileMode(0o644))
}

func TestRendererErrorOnConflictingFile(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	f, err := os.Create(filepath.Join(tmpDir, "a"))
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)

	r := renderer{
		skipPatterns: []string{},
		files: []file{
			&inMemoryFile{
				perm:    0o444,
				relPath: "a",
				content: []byte("123"),
			},
		},
	}
	out, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	err = r.persistToDisk(ctx, out)
	assert.EqualError(t, err, "failed to initialize template, one or more files already exist: "+"a")
}

func TestRendererNoErrorOnConflictingFileIfSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	f, err := os.Create(filepath.Join(tmpDir, "a"))
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)

	r := renderer{
		ctx:          ctx,
		skipPatterns: []string{"a"},
		files: []file{
			&inMemoryFile{
				perm:    0o444,
				relPath: "a",
				content: []byte("123"),
			},
		},
	}
	out, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	err = r.persistToDisk(ctx, out)
	// No error is returned even though a conflicting file exists. This is because
	// the generated file is being skipped
	assert.NoError(t, err)
	assert.Len(t, r.files, 1)
}

func TestRendererNonTemplatesAreCreatedAsCopyFiles(t *testing.T) {
	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, nil)

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/copy-file-walk/template", "./testdata/copy-file-walk/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	assert.Equal(t, "not-a-template", r.files[0].(*copyFile).srcPath)
	assert.Equal(t, "not-a-template", r.files[0].RelPath())
}

func TestRendererFileTreeRendering(t *testing.T) {
	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	tmpDir := t.TempDir()

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, map[string]any{
		"dir_name":  "my_directory",
		"file_name": "my_file",
	}, helpers, os.DirFS("."), "./testdata/file-tree-rendering/template", "./testdata/file-tree-rendering/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	// Assert in memory representation is created.
	assert.Len(t, r.files, 1)
	assert.Equal(t, "my_directory/my_file", r.files[0].RelPath())

	out, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	err = r.persistToDisk(ctx, out)
	require.NoError(t, err)

	// Assert files and directories are correctly materialized.
	testutil.AssertDirPermissions(t, filepath.Join(tmpDir, "my_directory"), defaultDirPermissions)
	testutil.AssertFilePermissions(t, filepath.Join(tmpDir, "my_directory", "my_file"), defaultFilePermissions)
}

func TestRendererSubTemplateInPath(t *testing.T) {
	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, nil)

	// Copy the template directory to a temporary directory where we can safely include a templated file path.
	// These paths include characters that are forbidden in Go modules, so we can't use the testdata directory.
	// Also see https://github.com/databricks/cli/pull/1671.
	templateDir := t.TempDir()
	testutil.CopyDirectory(t, "./testdata/template-in-path", templateDir)

	// Use a backtick-quoted string; double quotes are a reserved character for Windows paths:
	// https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file.
	testutil.Touch(t, filepath.Join(templateDir, "template/{{template `dir_name`}}/{{template `file_name`}}"))

	r, err := newRenderer(ctx, nil, nil, os.DirFS(templateDir), "template", "library")
	require.NoError(t, err)

	err = r.walk()
	require.NoError(t, err)

	if assert.Len(t, r.files, 2) {
		f := r.files[1]
		assert.Equal(t, "my_directory/my_file", f.RelPath())
	}
}
