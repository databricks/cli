package lsp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/spf13/cobra"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

const lsName = "databricks-lsp"

var version string = "0.0.1"
var handler protocol.Handler

type LspThingy interface {
	Match(uri protocol.DocumentUri) bool
}

type Linter interface {
	Lint(ctx context.Context, uri protocol.DocumentUri) ([]protocol.Diagnostic, error)
}

type QuickFixer interface {
	QuickFix(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error)
}

type LspMultiplexer struct {
	things []LspThingy
}

func (m *LspMultiplexer) Lint(ctx context.Context, uri protocol.DocumentUri) ([]protocol.Diagnostic, error) {
	diags := []protocol.Diagnostic{}
	for _, thing := range m.things {
		if !thing.Match(uri) {
			continue
		}
		linter, ok := thing.(Linter)
		if !ok {
			continue
		}
		problems, err := linter.Lint(ctx, uri)
		if err != nil {
			return nil, fmt.Errorf("linter: %w", err)
		}
		diags = append(diags, problems...)
	}
	return diags, nil
}

func (m *LspMultiplexer) QuickFix(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	actions := []protocol.CodeAction{}
	for _, thing := range m.things {
		if !thing.Match(params.TextDocument.URI) {
			continue
		}
		fixer, ok := thing.(QuickFixer)
		if !ok {
			continue
		}
		fixes, err := fixer.QuickFix(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("quick fixer: %w", err)
		}
		for _, fix := range fixes {
			actions = append(actions, fix)
		}
	}
	return actions, nil
}

type LocalLspProxy struct {
	host       string
	source     string
	extensions []string
	client     *httpclient.ApiClient
}

func (p *LocalLspProxy) Match(uri protocol.DocumentUri) bool {
	for _, ext := range p.extensions {
		if strings.HasSuffix(string(uri), ext) {
			return true
		}
	}
	return false
}

func (p *LocalLspProxy) Lint(ctx context.Context, uri protocol.DocumentUri) ([]protocol.Diagnostic, error) {
	var res struct {
		Diagnostics []protocol.Diagnostic `json:"diagnostics"`
	}
	err := p.client.Do(ctx, "GET", fmt.Sprintf("%s/lint", p.host),
		httpclient.WithRequestData(map[string]any{
			"file_uri": uri,
		}), httpclient.WithResponseUnmarshal(&res))
	if err != nil {
		return nil, err
	}
	return res.Diagnostics, nil
}

type FixMe struct {
	Range protocol.Range `json:"range"`
	Code  string         `json:"code"`

	resolves protocol.Diagnostic `json:"-"`
}

// match diagnostics produced by a given source
func (p *LocalLspProxy) matchDiagnostic(diagnostics []protocol.Diagnostic) *FixMe {
	for _, v := range diagnostics {
		if v.Source == nil {
			continue
		}
		if *v.Source != p.source {
			continue
		}
		if v.Code == nil {
			continue
		}
		return &FixMe{
			Range:    v.Range,
			Code:     fmt.Sprint(v.Code.Value),
			resolves: v,
		}
	}
	return nil
}

func (p *LocalLspProxy) QuickFix(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	fixMe := p.matchDiagnostic(params.Context.Diagnostics)
	if fixMe == nil {
		return nil, nil
	}
	var res struct {
		CodeActions []protocol.CodeAction `json:"code_actions"`
	}
	err := p.client.Do(ctx, "POST", fmt.Sprintf("%s/quickfix", p.host),
		httpclient.WithRequestData(map[string]any{
			"file_uri": params.TextDocument.URI,
			"code":     fixMe.Code,
			"range":    params.Range,
		}), httpclient.WithResponseUnmarshal(&res))
	if err != nil {
		return nil, err
	}
	// protocol.CodeActionKindSource has to be handled by a separate method, not QuickFix(...) - e.g reformatting
	quickFixKind := protocol.CodeActionKindQuickFix
	for i := range res.CodeActions {
		res.CodeActions[i].Diagnostics = []protocol.Diagnostic{fixMe.resolves}
		res.CodeActions[i].Kind = &quickFixKind
	}
	return res.CodeActions, nil
}

func startServer(ctx context.Context) error {
	commonlog.Configure(1, nil)

	// in production, we'll launch Databricks Labs command proxy, that
	// will return a JSON on stdout with the following structure:
	// {
	//   "host": "http://localhost:<random-port>",
	//   "source": "databricks.labs.<project-name>",
	//   "extensions": [".py", <other-extensions>]
	// }
	ucx := &LocalLspProxy{
		host:       "http://localhost:8000",
		source:     "databricks.labs.ucx",
		extensions: []string{".py", ".sql"},
		client:     httpclient.NewApiClient(httpclient.ClientConfig{}),
	}
	// and here we'll add DABs, DLT, linters, more SQL introspection, etc
	multiplexer := &LspMultiplexer{
		things: []LspThingy{ucx},
	}
	handler = protocol.Handler{
		Initialize:  initialize,
		Initialized: initialized,
		Shutdown:    shutdown,
		SetTrace:    setTrace,
		TextDocumentCodeAction: func(lsp *glsp.Context, params *protocol.CodeActionParams) (any, error) {
			started := time.Now()
			codeActions, err := multiplexer.QuickFix(ctx, params)
			protocol.Trace(lsp,
				protocol.MessageTypeLog,
				fmt.Sprintf("code action: %d items, range %v, took %s",
					len(codeActions),
					params.Range,
					time.Since(started).Round(time.Millisecond).String(),
				))
			return codeActions, err
		},
		TextDocumentDidOpen: func(lsp *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
			started := time.Now()
			problems, err := multiplexer.Lint(ctx, params.TextDocument.URI)
			if err != nil {
				return err
			}
			protocol.Trace(lsp,
				protocol.MessageTypeLog,
				fmt.Sprintf("did open: %d items: took %s",
					len(problems),
					time.Since(started).Round(time.Millisecond).String(),
				))
			if len(problems) == 0 {
				return nil
			}
			lsp.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
				URI:         params.TextDocument.URI,
				Diagnostics: problems,
			})
			return nil
		},
		TextDocumentDidChange: func(lsp *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
			started := time.Now()
			problems, err := multiplexer.Lint(ctx, params.TextDocument.URI)
			if err != nil {
				return err
			}
			protocol.Trace(lsp,
				protocol.MessageTypeLog,
				fmt.Sprintf("did open: %d items: took %s",
					len(problems),
					time.Since(started).Round(time.Millisecond).String(),
				))
			if len(problems) == 0 {
				return nil
			}
			lsp.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
				URI:         params.TextDocument.URI,
				Diagnostics: problems,
			})
			return nil
		},
	}

	server := server.NewServer(&handler, lsName, false)
	return server.RunWebSocket("127.0.0.1:12345")
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := handler.CreateServerCapabilities()
	protocol.SetTraceValue(protocol.TraceValueVerbose)

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
}

func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	return protocol.Trace(context, protocol.MessageTypeLog, "initialized")
}

func shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lsp",
		Args:  root.NoArgs,
		Short: "Start the databricks language server",
		Annotations: map[string]string{
			"template": "Databricks CLI v{{.Version}}\n",
		},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		return startServer(ctx)
	}

	return cmd
}
