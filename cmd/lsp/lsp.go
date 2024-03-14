package lsp

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

const lsName = "databricks-lsp"

var version string = "0.0.1"
var handler protocol.Handler

func startServer(ctx context.Context) error {

	commonlog.Configure(1, nil)

	handler = protocol.Handler{
		Initialize:  initialize,
		Initialized: initialized,
		Shutdown:    shutdown,
		SetTrace:    setTrace,
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
