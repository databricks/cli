package lsp

import (
	"context"

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

var localClient = httpclient.NewApiClient(httpclient.ClientConfig{})

type AnalyseResponse struct {
	Diagnostics []protocol.Diagnostic `json:"diagnostics"`
}

func callUcx(lspctx *glsp.Context, uri protocol.DocumentUri) error {
	var res AnalyseResponse
	err := localClient.Do(context.Background(), "GET", "http://localhost:8000/analyse",
		httpclient.WithRequestData(map[string]any{
			"file_uri": uri,
		}), httpclient.WithResponseUnmarshal(&res))
	if err != nil {
		return err
	}
 	lspctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: res.Diagnostics,
	})
	return nil
}

func startServer(ctx context.Context) error {
	commonlog.Configure(1, nil)

	handler = protocol.Handler{
		Initialize:  initialize,
		Initialized: initialized,
		Shutdown:    shutdown,
		SetTrace:    setTrace,
		TextDocumentCodeAction: func(context *glsp.Context, params *protocol.CodeActionParams) (any, error) {
			foundUcx := false
			var codeRange protocol.Range
			for _, v := range params.Context.Diagnostics {
				if v.Source == nil {
					continue
				}
				if *v.Source == "databricks.labs.ucx" {
					codeRange = v.Range
					foundUcx = true
				}
			}
			if !foundUcx {
				return nil, nil
			}
			quickFix := protocol.CodeActionKindQuickFix
			codeActions := []protocol.CodeAction{
				{
					Title: "Replace table with migrated table",
					Kind:  &quickFix,
					Edit: &protocol.WorkspaceEdit{
						DocumentChanges: []any{
							protocol.TextDocumentEdit{
								TextDocument: protocol.OptionalVersionedTextDocumentIdentifier{
									TextDocumentIdentifier: params.TextDocument,
								},
								Edits: []any{
									protocol.TextEdit{
										Range:   codeRange,
										NewText: "[beep-v3]",
									},
								},
							},
						},
					},
				},
			}
			return codeActions, nil
		},
		CodeActionResolve: func(context *glsp.Context, params *protocol.CodeAction) (*protocol.CodeAction, error) {
			return params, nil
		},
		TextDocumentDidOpen: func(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
			return callUcx(context, params.TextDocument.URI)
		},
		TextDocumentDidChange: func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
			return callUcx(context, params.TextDocument.URI)
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
