package workspace_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceList(t *testing.T) {
	ctx := context.Background()
	stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "workspace", "list", "/")
	outStr := stdout.String()
	assert.Contains(t, outStr, "ID")
	assert.Contains(t, outStr, "Type")
	assert.Contains(t, outStr, "Language")
	assert.Contains(t, outStr, "Path")
	assert.Equal(t, "", stderr.String())
}

func TestWorkpaceListErrorWhenNoArguments(t *testing.T) {
	ctx := context.Background()
	_, _, err := testcli.RequireErrorRun(t, ctx, "workspace", "list")
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
}

func TestWorkpaceGetStatusErrorWhenNoArguments(t *testing.T) {
	ctx := context.Background()
	_, _, err := testcli.RequireErrorRun(t, ctx, "workspace", "get-status")
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
}

func TestWorkpaceExportPrintsContents(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	tmpdir := acc.TemporaryWorkspaceDir(wt, "workspace-export-")
	f, err := filer.NewWorkspaceFilesClient(w, tmpdir)
	require.NoError(t, err)

	// Write file to workspace
	contents := "#!/usr/bin/bash\necho hello, world\n"
	err = f.Write(ctx, "file-a", strings.NewReader(contents))
	require.NoError(t, err)

	// Run export
	stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "workspace", "export", path.Join(tmpdir, "file-a"))
	assert.Equal(t, contents, stdout.String())
	assert.Equal(t, "", stderr.String())
}

func setupWorkspaceImportExportTest(t *testing.T) (context.Context, filer.Filer, string) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	tmpdir := acc.TemporaryWorkspaceDir(wt, "workspace-import-")
	f, err := filer.NewWorkspaceFilesClient(w, tmpdir)
	require.NoError(t, err)

	return ctx, f, tmpdir
}

func assertLocalFileContents(t *testing.T, path, content string) {
	require.FileExists(t, path)
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(b), content)
}

func assertFilerFileContents(t *testing.T, ctx context.Context, f filer.Filer, path, content string) {
	r, err := f.Read(ctx, path)
	require.NoError(t, err)
	b, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Contains(t, string(b), content)
}

func assertWorkspaceFileType(t *testing.T, ctx context.Context, f filer.Filer, path string, fileType workspace.ObjectType) {
	info, err := f.Stat(ctx, path)
	require.NoError(t, err)
	assert.Equal(t, fileType, info.Sys().(workspace.ObjectInfo).ObjectType)
}

func TestExportDir(t *testing.T) {
	ctx, f, sourceDir := setupWorkspaceImportExportTest(t)
	targetDir := t.TempDir()

	var err error

	// Write test data to the workspace
	err = f.Write(ctx, "file-a", strings.NewReader("abc"))
	require.NoError(t, err)
	err = f.Write(ctx, "pyNotebook.py", strings.NewReader("# Databricks notebook source"))
	require.NoError(t, err)
	err = f.Write(ctx, "sqlNotebook.sql", strings.NewReader("-- Databricks notebook source"))
	require.NoError(t, err)
	err = f.Write(ctx, "scalaNotebook.scala", strings.NewReader("// Databricks notebook source"))
	require.NoError(t, err)
	err = f.Write(ctx, "rNotebook.r", strings.NewReader("# Databricks notebook source"))
	require.NoError(t, err)
	err = f.Write(ctx, "a/b/c/file-b", strings.NewReader("def"), filer.CreateParentDirectories)
	require.NoError(t, err)

	expectedLogs := strings.Join([]string{
		"Exporting files from " + sourceDir,
		fmt.Sprintf("%s -> %s", path.Join(sourceDir, "a/b/c/file-b"), filepath.Join(targetDir, "a/b/c/file-b")),
		fmt.Sprintf("%s -> %s", path.Join(sourceDir, "file-a"), filepath.Join(targetDir, "file-a")),
		fmt.Sprintf("%s -> %s", path.Join(sourceDir, "pyNotebook"), filepath.Join(targetDir, "pyNotebook.py")),
		fmt.Sprintf("%s -> %s", path.Join(sourceDir, "rNotebook"), filepath.Join(targetDir, "rNotebook.r")),
		fmt.Sprintf("%s -> %s", path.Join(sourceDir, "scalaNotebook"), filepath.Join(targetDir, "scalaNotebook.scala")),
		fmt.Sprintf("%s -> %s", path.Join(sourceDir, "sqlNotebook"), filepath.Join(targetDir, "sqlNotebook.sql")),
		"Export complete\n",
	}, "\n")

	// Run Export
	stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "workspace", "export-dir", sourceDir, targetDir)
	assert.Equal(t, expectedLogs, stdout.String())
	assert.Equal(t, "", stderr.String())

	// Assert files were exported
	assertLocalFileContents(t, filepath.Join(targetDir, "file-a"), "abc")
	assertLocalFileContents(t, filepath.Join(targetDir, "pyNotebook.py"), "# Databricks notebook source")
	assertLocalFileContents(t, filepath.Join(targetDir, "sqlNotebook.sql"), "-- Databricks notebook source")
	assertLocalFileContents(t, filepath.Join(targetDir, "rNotebook.r"), "# Databricks notebook source")
	assertLocalFileContents(t, filepath.Join(targetDir, "scalaNotebook.scala"), "// Databricks notebook source")
	assertLocalFileContents(t, filepath.Join(targetDir, "a/b/c/file-b"), "def")
}

func TestExportDirDoesNotOverwrite(t *testing.T) {
	ctx, f, sourceDir := setupWorkspaceImportExportTest(t)
	targetDir := t.TempDir()

	var err error

	// Write remote file
	err = f.Write(ctx, "file-a", strings.NewReader("content from workspace"))
	require.NoError(t, err)

	// Write local file
	err = os.WriteFile(filepath.Join(targetDir, "file-a"), []byte("local content"), os.ModePerm)
	require.NoError(t, err)

	// Run Export
	testcli.RequireSuccessfulRun(t, ctx, "workspace", "export-dir", sourceDir, targetDir)

	// Assert file is not overwritten
	assertLocalFileContents(t, filepath.Join(targetDir, "file-a"), "local content")
}

func TestExportDirWithOverwriteFlag(t *testing.T) {
	ctx, f, sourceDir := setupWorkspaceImportExportTest(t)
	targetDir := t.TempDir()

	var err error

	// Write remote file
	err = f.Write(ctx, "file-a", strings.NewReader("content from workspace"))
	require.NoError(t, err)

	// Write local file
	err = os.WriteFile(filepath.Join(targetDir, "file-a"), []byte("local content"), os.ModePerm)
	require.NoError(t, err)

	// Run Export
	testcli.RequireSuccessfulRun(t, ctx, "workspace", "export-dir", sourceDir, targetDir, "--overwrite")

	// Assert file has been overwritten
	assertLocalFileContents(t, filepath.Join(targetDir, "file-a"), "content from workspace")
}

func TestImportDir(t *testing.T) {
	ctx, workspaceFiler, targetDir := setupWorkspaceImportExportTest(t)
	stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "workspace", "import-dir", "./testdata/import_dir", targetDir, "--log-level=debug")

	expectedLogs := strings.Join([]string{
		"Importing files from " + "./testdata/import_dir",
		fmt.Sprintf("%s -> %s", filepath.FromSlash("a/b/c/file-b"), path.Join(targetDir, "a/b/c/file-b")),
		fmt.Sprintf("%s -> %s", filepath.FromSlash("file-a"), path.Join(targetDir, "file-a")),
		fmt.Sprintf("%s -> %s", filepath.FromSlash("jupyterNotebook.ipynb"), path.Join(targetDir, "jupyterNotebook")),
		fmt.Sprintf("%s -> %s", filepath.FromSlash("pyNotebook.py"), path.Join(targetDir, "pyNotebook")),
		fmt.Sprintf("%s -> %s", filepath.FromSlash("rNotebook.r"), path.Join(targetDir, "rNotebook")),
		fmt.Sprintf("%s -> %s", filepath.FromSlash("scalaNotebook.scala"), path.Join(targetDir, "scalaNotebook")),
		fmt.Sprintf("%s -> %s", filepath.FromSlash("sqlNotebook.sql"), path.Join(targetDir, "sqlNotebook")),
		"Import complete\n",
	}, "\n")

	assert.Equal(t, expectedLogs, stdout.String())
	assert.Equal(t, "", stderr.String())

	// Assert files are imported
	assertFilerFileContents(t, ctx, workspaceFiler, "file-a", "hello, world")
	assertFilerFileContents(t, ctx, workspaceFiler, "a/b/c/file-b", "file-in-dir")
	assertFilerFileContents(t, ctx, workspaceFiler, "pyNotebook", "# Databricks notebook source\nprint(\"python\")")
	assertFilerFileContents(t, ctx, workspaceFiler, "sqlNotebook", "-- Databricks notebook source\nSELECT \"sql\"")
	assertFilerFileContents(t, ctx, workspaceFiler, "rNotebook", "# Databricks notebook source\nprint(\"r\")")
	assertFilerFileContents(t, ctx, workspaceFiler, "scalaNotebook", "// Databricks notebook source\nprintln(\"scala\")")
	assertFilerFileContents(t, ctx, workspaceFiler, "jupyterNotebook", "# Databricks notebook source\nprint(\"jupyter\")")
}

func TestImportDirDoesNotOverwrite(t *testing.T) {
	ctx, workspaceFiler, targetDir := setupWorkspaceImportExportTest(t)
	var err error

	// create preexisting files in the workspace
	err = workspaceFiler.Write(ctx, "file-a", strings.NewReader("old file"))
	require.NoError(t, err)
	err = workspaceFiler.Write(ctx, "pyNotebook.py", strings.NewReader("# Databricks notebook source\nprint(\"old notebook\")"))
	require.NoError(t, err)

	// Assert contents of pre existing files
	assertFilerFileContents(t, ctx, workspaceFiler, "file-a", "old file")
	assertFilerFileContents(t, ctx, workspaceFiler, "pyNotebook", "# Databricks notebook source\nprint(\"old notebook\")")

	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import-dir", "./testdata/import_dir", targetDir)

	// Assert files are imported
	assertFilerFileContents(t, ctx, workspaceFiler, "a/b/c/file-b", "file-in-dir")
	assertFilerFileContents(t, ctx, workspaceFiler, "sqlNotebook", "-- Databricks notebook source\nSELECT \"sql\"")
	assertFilerFileContents(t, ctx, workspaceFiler, "rNotebook", "# Databricks notebook source\nprint(\"r\")")
	assertFilerFileContents(t, ctx, workspaceFiler, "scalaNotebook", "// Databricks notebook source\nprintln(\"scala\")")
	assertFilerFileContents(t, ctx, workspaceFiler, "jupyterNotebook", "# Databricks notebook source\nprint(\"jupyter\")")

	// Assert pre existing files are not changed
	assertFilerFileContents(t, ctx, workspaceFiler, "file-a", "old file")
	assertFilerFileContents(t, ctx, workspaceFiler, "pyNotebook", "# Databricks notebook source\nprint(\"old notebook\")")
}

func TestImportDirWithOverwriteFlag(t *testing.T) {
	ctx, workspaceFiler, targetDir := setupWorkspaceImportExportTest(t)
	var err error

	// create preexisting files in the workspace
	err = workspaceFiler.Write(ctx, "file-a", strings.NewReader("old file"))
	require.NoError(t, err)
	err = workspaceFiler.Write(ctx, "pyNotebook.py", strings.NewReader("# Databricks notebook source\nprint(\"old notebook\")"))
	require.NoError(t, err)

	// Assert contents of pre existing files
	assertFilerFileContents(t, ctx, workspaceFiler, "file-a", "old file")
	assertFilerFileContents(t, ctx, workspaceFiler, "pyNotebook", "# Databricks notebook source\nprint(\"old notebook\")")

	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import-dir", "./testdata/import_dir", targetDir, "--overwrite")

	// Assert files are imported
	assertFilerFileContents(t, ctx, workspaceFiler, "a/b/c/file-b", "file-in-dir")
	assertFilerFileContents(t, ctx, workspaceFiler, "sqlNotebook", "-- Databricks notebook source\nSELECT \"sql\"")
	assertFilerFileContents(t, ctx, workspaceFiler, "rNotebook", "# Databricks notebook source\nprint(\"r\")")
	assertFilerFileContents(t, ctx, workspaceFiler, "scalaNotebook", "// Databricks notebook source\nprintln(\"scala\")")
	assertFilerFileContents(t, ctx, workspaceFiler, "jupyterNotebook", "# Databricks notebook source\nprint(\"jupyter\")")

	// Assert pre existing files are overwritten
	assertFilerFileContents(t, ctx, workspaceFiler, "file-a", "hello, world")
	assertFilerFileContents(t, ctx, workspaceFiler, "pyNotebook", "# Databricks notebook source\nprint(\"python\")")
}

func TestExport(t *testing.T) {
	ctx, f, sourceDir := setupWorkspaceImportExportTest(t)

	var err error

	// Export vanilla file
	err = f.Write(ctx, "file-a", strings.NewReader("abc"))
	require.NoError(t, err)
	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "workspace", "export", path.Join(sourceDir, "file-a"))
	b, err := io.ReadAll(&stdout)
	require.NoError(t, err)
	assert.Equal(t, "abc", string(b))

	// Export python notebook
	err = f.Write(ctx, "pyNotebook.py", strings.NewReader("# Databricks notebook source"))
	require.NoError(t, err)
	stdout, _ = testcli.RequireSuccessfulRun(t, ctx, "workspace", "export", path.Join(sourceDir, "pyNotebook"))
	b, err = io.ReadAll(&stdout)
	require.NoError(t, err)
	assert.Equal(t, "# Databricks notebook source\n", string(b))

	// Export python notebook as jupyter
	stdout, _ = testcli.RequireSuccessfulRun(t, ctx, "workspace", "export", path.Join(sourceDir, "pyNotebook"), "--format", "JUPYTER")
	b, err = io.ReadAll(&stdout)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"cells":`, "jupyter notebooks contain the cells field")
	assert.Contains(t, string(b), `"metadata":`, "jupyter notebooks contain the metadata field")
}

func TestExportWithFileFlag(t *testing.T) {
	ctx, f, sourceDir := setupWorkspaceImportExportTest(t)
	localTmpDir := t.TempDir()

	var err error

	// Export vanilla file
	err = f.Write(ctx, "file-a", strings.NewReader("abc"))
	require.NoError(t, err)
	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "workspace", "export", path.Join(sourceDir, "file-a"), "--file", filepath.Join(localTmpDir, "file.txt"))
	b, err := io.ReadAll(&stdout)
	require.NoError(t, err)
	// Expect nothing to be printed to stdout
	assert.Equal(t, "", string(b))
	assertLocalFileContents(t, filepath.Join(localTmpDir, "file.txt"), "abc")

	// Export python notebook
	err = f.Write(ctx, "pyNotebook.py", strings.NewReader("# Databricks notebook source"))
	require.NoError(t, err)
	stdout, _ = testcli.RequireSuccessfulRun(t, ctx, "workspace", "export", path.Join(sourceDir, "pyNotebook"), "--file", filepath.Join(localTmpDir, "pyNb.py"))
	b, err = io.ReadAll(&stdout)
	require.NoError(t, err)
	assert.Equal(t, "", string(b))
	assertLocalFileContents(t, filepath.Join(localTmpDir, "pyNb.py"), "# Databricks notebook source\n")

	// Export python notebook as jupyter
	stdout, _ = testcli.RequireSuccessfulRun(t, ctx, "workspace", "export", path.Join(sourceDir, "pyNotebook"), "--format", "JUPYTER", "--file", filepath.Join(localTmpDir, "jupyterNb.ipynb"))
	b, err = io.ReadAll(&stdout)
	require.NoError(t, err)
	assert.Equal(t, "", string(b))
	assertLocalFileContents(t, filepath.Join(localTmpDir, "jupyterNb.ipynb"), `"cells":`)
	assertLocalFileContents(t, filepath.Join(localTmpDir, "jupyterNb.ipynb"), `"metadata":`)
}

func TestImportFileUsingContentFormatSource(t *testing.T) {
	ctx, workspaceFiler, targetDir := setupWorkspaceImportExportTest(t)

	//  Content = `print(1)`. Uploaded as a notebook by default
	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import", path.Join(targetDir, "pyScript"),
		"--content", base64.StdEncoding.EncodeToString([]byte("print(1)")), "--language=PYTHON")
	assertFilerFileContents(t, ctx, workspaceFiler, "pyScript", "print(1)")
	assertWorkspaceFileType(t, ctx, workspaceFiler, "pyScript", workspace.ObjectTypeNotebook)

	// Import with content = `# Databricks notebook source\nprint(1)`. Uploaded as a notebook with the content just being print(1)
	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import", path.Join(targetDir, "pyNb"),
		"--content", base64.StdEncoding.EncodeToString([]byte("`# Databricks notebook source\nprint(1)")),
		"--language=PYTHON")
	assertFilerFileContents(t, ctx, workspaceFiler, "pyNb", "print(1)")
	assertWorkspaceFileType(t, ctx, workspaceFiler, "pyNb", workspace.ObjectTypeNotebook)
}

func TestImportFileUsingContentFormatAuto(t *testing.T) {
	ctx, workspaceFiler, targetDir := setupWorkspaceImportExportTest(t)

	//  Content = `# Databricks notebook source\nprint(1)`. Upload as file if path has no extension.
	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import", path.Join(targetDir, "py-nb-as-file"),
		"--content", base64.StdEncoding.EncodeToString([]byte("`# Databricks notebook source\nprint(1)")), "--format=AUTO")
	assertFilerFileContents(t, ctx, workspaceFiler, "py-nb-as-file", "# Databricks notebook source\nprint(1)")
	assertWorkspaceFileType(t, ctx, workspaceFiler, "py-nb-as-file", workspace.ObjectTypeFile)

	// Content = `# Databricks notebook source\nprint(1)`. Upload as notebook if path has py extension
	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import", path.Join(targetDir, "py-nb-as-notebook.py"),
		"--content", base64.StdEncoding.EncodeToString([]byte("`# Databricks notebook source\nprint(1)")), "--format=AUTO")
	assertFilerFileContents(t, ctx, workspaceFiler, "py-nb-as-notebook", "# Databricks notebook source\nprint(1)")
	assertWorkspaceFileType(t, ctx, workspaceFiler, "py-nb-as-notebook", workspace.ObjectTypeNotebook)

	// Content = `print(1)`. Upload as file if content is not notebook (even if path has .py extension)
	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import", path.Join(targetDir, "not-a-notebook.py"), "--content",
		base64.StdEncoding.EncodeToString([]byte("print(1)")), "--format=AUTO")
	assertFilerFileContents(t, ctx, workspaceFiler, "not-a-notebook.py", "print(1)")
	assertWorkspaceFileType(t, ctx, workspaceFiler, "not-a-notebook.py", workspace.ObjectTypeFile)
}

func TestImportFileFormatSource(t *testing.T) {
	ctx, workspaceFiler, targetDir := setupWorkspaceImportExportTest(t)
	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import", path.Join(targetDir, "pyNotebook"), "--file", "./testdata/import_dir/pyNotebook.py", "--language=PYTHON")
	assertFilerFileContents(t, ctx, workspaceFiler, "pyNotebook", "# Databricks notebook source\nprint(\"python\")")
	assertWorkspaceFileType(t, ctx, workspaceFiler, "pyNotebook", workspace.ObjectTypeNotebook)

	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import", path.Join(targetDir, "scalaNotebook"), "--file", "./testdata/import_dir/scalaNotebook.scala", "--language=SCALA")
	assertFilerFileContents(t, ctx, workspaceFiler, "scalaNotebook", "// Databricks notebook source\nprintln(\"scala\")")
	assertWorkspaceFileType(t, ctx, workspaceFiler, "scalaNotebook", workspace.ObjectTypeNotebook)

	_, _, err := testcli.RequireErrorRun(t, ctx, "workspace", "import", path.Join(targetDir, "scalaNotebook"), "--file", "./testdata/import_dir/scalaNotebook.scala")
	assert.ErrorContains(t, err, "The zip file may not be valid or may be an unsupported version. Hint: Objects imported using format=SOURCE are expected to be zip encoded databricks source notebook(s) by default. Please specify a language using the --language flag if you are trying to import a single uncompressed notebook")
}

func TestImportFileFormatAuto(t *testing.T) {
	ctx, workspaceFiler, targetDir := setupWorkspaceImportExportTest(t)

	// Upload as file if path has no extension
	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import", path.Join(targetDir, "py-nb-as-file"), "--file", "./testdata/import_dir/pyNotebook.py", "--format=AUTO")
	assertFilerFileContents(t, ctx, workspaceFiler, "py-nb-as-file", "# Databricks notebook source")
	assertFilerFileContents(t, ctx, workspaceFiler, "py-nb-as-file", "print(\"python\")")
	assertWorkspaceFileType(t, ctx, workspaceFiler, "py-nb-as-file", workspace.ObjectTypeFile)

	// Upload as notebook if path has extension
	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import", path.Join(targetDir, "py-nb-as-notebook.py"), "--file", "./testdata/import_dir/pyNotebook.py", "--format=AUTO")
	assertFilerFileContents(t, ctx, workspaceFiler, "py-nb-as-notebook", "# Databricks notebook source\nprint(\"python\")")
	assertWorkspaceFileType(t, ctx, workspaceFiler, "py-nb-as-notebook", workspace.ObjectTypeNotebook)

	// Upload as file if content is not notebook (even if path has .py extension)
	testcli.RequireSuccessfulRun(t, ctx, "workspace", "import", path.Join(targetDir, "not-a-notebook.py"), "--file", "./testdata/import_dir/file-a", "--format=AUTO")
	assertFilerFileContents(t, ctx, workspaceFiler, "not-a-notebook.py", "hello, world")
	assertWorkspaceFileType(t, ctx, workspaceFiler, "not-a-notebook.py", workspace.ObjectTypeFile)
}
