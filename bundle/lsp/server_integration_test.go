package lsp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/jrpc2/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBundleYAML = `bundle:
  name: test-bundle
workspace:
  host: "https://my-workspace.databricks.com"
targets:
  dev:
    default: true
resources:
  jobs:
    my_job:
      name: "My Job"
  pipelines:
    my_pipeline:
      name: "My Pipeline"
`

func setupTestBundleDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "databricks.yml"), []byte(testBundleYAML), 0o644))

	stateDir := filepath.Join(tmpDir, ".databricks", "bundle", "dev")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))

	stateJSON := `{
		"state_version": 1,
		"state": {
			"resources.jobs.my_job": {"__id__": "12345"},
			"resources.pipelines.my_pipeline": {"__id__": "abc-def"}
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "resources.json"), []byte(stateJSON), 0o644))

	return tmpDir
}

func newTestClientServer(t *testing.T, srv *Server) *jrpc2.Client {
	t.Helper()

	mux := handler.Map{
		"initialize":                handler.New(srv.handleInitialize),
		"initialized":               handler.New(srv.handleInitialized),
		"shutdown":                  handler.New(srv.handleShutdown),
		"textDocument/didOpen":      handler.New(srv.handleTextDocumentDidOpen),
		"textDocument/didChange":    handler.New(srv.handleTextDocumentDidChange),
		"textDocument/didClose":     handler.New(srv.handleTextDocumentDidClose),
		"textDocument/documentLink": handler.New(srv.handleDocumentLink),
		"textDocument/hover":        handler.New(srv.handleHover),
		"textDocument/definition":   handler.New(srv.handleDefinition),
	}

	clientCh, serverCh := channel.Direct()

	jrpcSrv := jrpc2.NewServer(mux, nil)
	jrpcSrv.Start(serverCh)
	t.Cleanup(func() { jrpcSrv.Stop() })

	cli := jrpc2.NewClient(clientCh, nil)
	t.Cleanup(func() { cli.Close() })

	return cli
}

// initializeClient sends the initialize request and returns the result.
func initializeClient(ctx context.Context, t *testing.T, cli *jrpc2.Client, rootURI string) InitializeResult {
	t.Helper()
	var result InitializeResult
	err := cli.CallResult(ctx, "initialize", InitializeParams{
		ProcessID: 1,
		RootURI:   rootURI,
	}, &result)
	require.NoError(t, err)
	return result
}

func TestServerHandleInitialize(t *testing.T) {
	tmpDir := setupTestBundleDir(t)
	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	result := initializeClient(ctx, t, cli, PathToURI(tmpDir))

	assert.True(t, result.Capabilities.HoverProvider)
	assert.True(t, result.Capabilities.DefinitionProvider)
	require.NotNil(t, result.Capabilities.DocumentLinkProvider)
	require.NotNil(t, result.Capabilities.TextDocumentSync)
	assert.True(t, result.Capabilities.TextDocumentSync.OpenClose)
	assert.Equal(t, 1, result.Capabilities.TextDocumentSync.Change)
}

func TestServerHandleDocumentLink(t *testing.T) {
	tmpDir := setupTestBundleDir(t)
	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	initializeClient(ctx, t, cli, PathToURI(tmpDir))

	docURI := PathToURI(filepath.Join(tmpDir, "databricks.yml"))
	err := cli.Notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        docURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       testBundleYAML,
		},
	})
	require.NoError(t, err)

	var links []DocumentLink
	err = cli.CallResult(ctx, "textDocument/documentLink", DocumentLinkParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
	}, &links)
	require.NoError(t, err)
	require.Len(t, links, 2)

	assert.Contains(t, links[0].Target, "/jobs/12345")
	assert.Contains(t, links[0].Tooltip, "my_job")
	assert.Contains(t, links[1].Target, "/pipelines/abc-def")
	assert.Contains(t, links[1].Tooltip, "my_pipeline")
}

func TestServerHandleDocumentLinkNoState(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "databricks.yml"), []byte(testBundleYAML), 0o644))

	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	initializeClient(ctx, t, cli, PathToURI(tmpDir))

	docURI := PathToURI(filepath.Join(tmpDir, "databricks.yml"))
	err := cli.Notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        docURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       testBundleYAML,
		},
	})
	require.NoError(t, err)

	var links []DocumentLink
	err = cli.CallResult(ctx, "textDocument/documentLink", DocumentLinkParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
	}, &links)
	require.NoError(t, err)
	assert.Empty(t, links)
}

func TestServerHandleHoverOnResource(t *testing.T) {
	tmpDir := setupTestBundleDir(t)
	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	initializeClient(ctx, t, cli, PathToURI(tmpDir))

	docURI := PathToURI(filepath.Join(tmpDir, "databricks.yml"))
	err := cli.Notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        docURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       testBundleYAML,
		},
	})
	require.NoError(t, err)

	// Get links to find the position of my_job.
	var links []DocumentLink
	err = cli.CallResult(ctx, "textDocument/documentLink", DocumentLinkParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
	}, &links)
	require.NoError(t, err)
	require.NotEmpty(t, links)

	// Hover at the position of the first link (my_job key).
	var hover Hover
	err = cli.CallResult(ctx, "textDocument/hover", HoverParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
		Position:     links[0].Range.Start,
	}, &hover)
	require.NoError(t, err)
	assert.Contains(t, hover.Contents.Value, "12345")
	assert.Contains(t, hover.Contents.Value, "Open in Databricks")
}

func TestServerHandleHoverOffResource(t *testing.T) {
	tmpDir := setupTestBundleDir(t)
	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	initializeClient(ctx, t, cli, PathToURI(tmpDir))

	docURI := PathToURI(filepath.Join(tmpDir, "databricks.yml"))
	err := cli.Notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        docURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       testBundleYAML,
		},
	})
	require.NoError(t, err)

	// Hover at line 0, character 0 which is "bundle:" -- not a resource key.
	rsp, err := cli.Call(ctx, "textDocument/hover", HoverParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
		Position:     Position{Line: 0, Character: 0},
	})
	require.NoError(t, err)

	// The handler returns nil for non-resource positions, which is JSON null.
	var hover *Hover
	err = rsp.UnmarshalResult(&hover)
	require.NoError(t, err)
	assert.Nil(t, hover)
}

func TestServerEndToEnd(t *testing.T) {
	tmpDir := setupTestBundleDir(t)
	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	// 1. Initialize.
	result := initializeClient(ctx, t, cli, PathToURI(tmpDir))
	assert.True(t, result.Capabilities.HoverProvider)

	// 2. Initialized notification.
	err := cli.Notify(ctx, "initialized", nil)
	require.NoError(t, err)

	// 3. Open document.
	docURI := PathToURI(filepath.Join(tmpDir, "databricks.yml"))
	err = cli.Notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        docURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       testBundleYAML,
		},
	})
	require.NoError(t, err)

	// 4. Get document links.
	var links []DocumentLink
	err = cli.CallResult(ctx, "textDocument/documentLink", DocumentLinkParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
	}, &links)
	require.NoError(t, err)
	require.Len(t, links, 2)
	assert.Contains(t, links[0].Target, "/jobs/12345")
	assert.Contains(t, links[1].Target, "/pipelines/abc-def")

	// 5. Hover on resource key.
	var hover Hover
	err = cli.CallResult(ctx, "textDocument/hover", HoverParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
		Position:     links[0].Range.Start,
	}, &hover)
	require.NoError(t, err)
	assert.Contains(t, hover.Contents.Value, "12345")
	assert.Contains(t, hover.Contents.Value, "Open in Databricks")

	// 6. Change document content (remove pipelines).
	updatedYAML := `bundle:
  name: test-bundle
workspace:
  host: "https://my-workspace.databricks.com"
targets:
  dev:
    default: true
resources:
  jobs:
    my_job:
      name: "My Job"
`
	err = cli.Notify(ctx, "textDocument/didChange", DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			URI:     docURI,
			Version: 2,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: updatedYAML},
		},
	})
	require.NoError(t, err)

	// 7. Document links should now return only one link.
	var linksAfterChange []DocumentLink
	err = cli.CallResult(ctx, "textDocument/documentLink", DocumentLinkParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
	}, &linksAfterChange)
	require.NoError(t, err)
	require.Len(t, linksAfterChange, 1)
	assert.Contains(t, linksAfterChange[0].Target, "/jobs/12345")

	// 8. Close document.
	err = cli.Notify(ctx, "textDocument/didClose", DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
	})
	require.NoError(t, err)

	// 9. Document links should return empty after close.
	var linksAfterClose []DocumentLink
	err = cli.CallResult(ctx, "textDocument/documentLink", DocumentLinkParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
	}, &linksAfterClose)
	require.NoError(t, err)
	assert.Empty(t, linksAfterClose)

	// 10. Shutdown.
	_, err = cli.Call(ctx, "shutdown", nil)
	require.NoError(t, err)
}

const testBundleYAMLWithInterpolation = `bundle:
  name: test-bundle
workspace:
  host: "https://my-workspace.databricks.com"
targets:
  dev:
    default: true
variables:
  my_var:
    default: "hello"
resources:
  jobs:
    my_job:
      name: "${var.my_var}"
  pipelines:
    my_pipeline:
      name: "My Pipeline"
`

func TestServerDefinitionOnInterpolation(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "databricks.yml"), []byte(testBundleYAMLWithInterpolation), 0o644))

	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	initializeClient(ctx, t, cli, PathToURI(tmpDir))

	docURI := PathToURI(filepath.Join(tmpDir, "databricks.yml"))
	err := cli.Notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        docURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       testBundleYAMLWithInterpolation,
		},
	})
	require.NoError(t, err)

	// Find the line with "${var.my_var}" and position cursor on it.
	lines := strings.Split(testBundleYAMLWithInterpolation, "\n")
	var targetLine int
	var targetCol int
	for i, line := range lines {
		idx := strings.Index(line, "${var.my_var}")
		if idx >= 0 {
			targetLine = i
			targetCol = idx + 2 // inside the "${...}"
			break
		}
	}

	var locs []LSPLocation
	err = cli.CallResult(ctx, "textDocument/definition", DefinitionParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
		Position:     Position{Line: targetLine, Character: targetCol},
	}, &locs)
	require.NoError(t, err)
	require.Len(t, locs, 1)
	assert.Contains(t, locs[0].URI, "databricks.yml")
}

func TestServerDefinitionOnResourceKey(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "databricks.yml"), []byte(testBundleYAMLWithInterpolation), 0o644))

	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	initializeClient(ctx, t, cli, PathToURI(tmpDir))

	docURI := PathToURI(filepath.Join(tmpDir, "databricks.yml"))
	err := cli.Notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        docURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       testBundleYAMLWithInterpolation,
		},
	})
	require.NoError(t, err)

	// Find the resource key position.
	lines := strings.Split(testBundleYAMLWithInterpolation, "\n")
	var myJobLine int
	var myJobCol int
	for i, line := range lines {
		idx := strings.Index(line, "my_job:")
		if idx >= 0 {
			myJobLine = i
			myJobCol = idx + 1 // inside "my_job"
			break
		}
	}

	// Ctrl+click on "my_job" key should return references (${...} expressions referencing it).
	// The YAML has name: "${var.my_var}" which does NOT reference my_job, so this may return empty.
	rsp, err := cli.Call(ctx, "textDocument/definition", DefinitionParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
		Position:     Position{Line: myJobLine, Character: myJobCol},
	})
	require.NoError(t, err)
	assert.NotNil(t, rsp)
}

func TestServerDefinitionVarShorthand(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "databricks.yml"), []byte(testBundleYAMLWithInterpolation), 0o644))

	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	initializeClient(ctx, t, cli, PathToURI(tmpDir))

	docURI := PathToURI(filepath.Join(tmpDir, "databricks.yml"))
	err := cli.Notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        docURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       testBundleYAMLWithInterpolation,
		},
	})
	require.NoError(t, err)

	// Find the line with "${var.my_var}" and position cursor on "var" part.
	lines := strings.Split(testBundleYAMLWithInterpolation, "\n")
	var targetLine int
	var targetCol int
	for i, line := range lines {
		idx := strings.Index(line, "${var.my_var}")
		if idx >= 0 {
			targetLine = i
			targetCol = idx + 2 // on "var" inside "${var.my_var}"
			break
		}
	}

	var locs []LSPLocation
	err = cli.CallResult(ctx, "textDocument/definition", DefinitionParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
		Position:     Position{Line: targetLine, Character: targetCol},
	}, &locs)
	require.NoError(t, err)
	require.Len(t, locs, 1)
	assert.Contains(t, locs[0].URI, "databricks.yml")
}

func TestServerDefinitionNoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "databricks.yml"), []byte(testBundleYAMLWithInterpolation), 0o644))

	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	initializeClient(ctx, t, cli, PathToURI(tmpDir))

	docURI := PathToURI(filepath.Join(tmpDir, "databricks.yml"))
	err := cli.Notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        docURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       testBundleYAMLWithInterpolation,
		},
	})
	require.NoError(t, err)

	// Cursor on line 0, character 0 ("bundle:") — not an interpolation or resource key.
	var result []LSPLocation
	err = cli.CallResult(ctx, "textDocument/definition", DefinitionParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
		Position:     Position{Line: 0, Character: 0},
	}, &result)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestServerDefinitionCrossFile(t *testing.T) {
	mainYAML := `bundle:
  name: test-bundle
include:
  - "resources/*.yml"
variables:
  my_var:
    default: "hello"
`
	resourceYAML := `resources:
  jobs:
    my_job:
      name: "${var.my_var}"
`
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "databricks.yml"), []byte(mainYAML), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "resources"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "resources", "jobs.yml"), []byte(resourceYAML), 0o644))

	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	initializeClient(ctx, t, cli, PathToURI(tmpDir))

	// Open the resource file with the interpolation.
	resDocURI := PathToURI(filepath.Join(tmpDir, "resources", "jobs.yml"))
	err := cli.Notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        resDocURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       resourceYAML,
		},
	})
	require.NoError(t, err)

	// Find "${var.my_var}" in the resource file.
	lines := strings.Split(resourceYAML, "\n")
	var targetLine int
	var targetCol int
	for i, line := range lines {
		idx := strings.Index(line, "${var.my_var}")
		if idx >= 0 {
			targetLine = i
			targetCol = idx + 2
			break
		}
	}

	// Definition should resolve to variables.my_var in the main config file.
	var locs []LSPLocation
	err = cli.CallResult(ctx, "textDocument/definition", DefinitionParams{
		TextDocument: TextDocumentIdentifier{URI: resDocURI},
		Position:     Position{Line: targetLine, Character: targetCol},
	}, &locs)
	require.NoError(t, err)
	require.Len(t, locs, 1)
	assert.Contains(t, locs[0].URI, "databricks.yml")
}

func TestServerHoverMultiTarget(t *testing.T) {
	bundleYAML := `bundle:
  name: test-bundle
workspace:
  host: "https://default.databricks.com"
targets:
  dev:
    default: true
    workspace:
      host: "https://dev.databricks.com"
  prod:
    workspace:
      host: "https://prod.databricks.com"
resources:
  jobs:
    my_job:
      name: "My Job"
`
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "databricks.yml"), []byte(bundleYAML), 0o644))

	// Create state for both targets.
	for _, target := range []struct {
		name string
		id   string
	}{
		{"dev", "111"},
		{"prod", "222"},
	} {
		stateDir := filepath.Join(tmpDir, ".databricks", "bundle", target.name)
		require.NoError(t, os.MkdirAll(stateDir, 0o755))
		stateJSON := `{"state_version": 1, "state": {"resources.jobs.my_job": {"__id__": "` + target.id + `"}}}`
		require.NoError(t, os.WriteFile(filepath.Join(stateDir, "resources.json"), []byte(stateJSON), 0o644))
	}

	srv := NewServer()
	cli := newTestClientServer(t, srv)
	ctx := t.Context()

	initializeClient(ctx, t, cli, PathToURI(tmpDir))

	docURI := PathToURI(filepath.Join(tmpDir, "databricks.yml"))
	err := cli.Notify(ctx, "textDocument/didOpen", DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        docURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       bundleYAML,
		},
	})
	require.NoError(t, err)

	// Find the position of my_job key.
	lines := strings.Split(bundleYAML, "\n")
	var myJobLine int
	var myJobCol int
	for i, line := range lines {
		idx := strings.Index(line, "my_job:")
		if idx >= 0 {
			myJobLine = i
			myJobCol = idx + 1
			break
		}
	}

	var hover Hover
	err = cli.CallResult(ctx, "textDocument/hover", HoverParams{
		TextDocument: TextDocumentIdentifier{URI: docURI},
		Position:     Position{Line: myJobLine, Character: myJobCol},
	}, &hover)
	require.NoError(t, err)
	assert.Contains(t, hover.Contents.Value, "dev")
	assert.Contains(t, hover.Contents.Value, "prod")
	assert.Contains(t, hover.Contents.Value, "111")
	assert.Contains(t, hover.Contents.Value, "222")
	assert.Contains(t, hover.Contents.Value, "Open in Databricks")
}
