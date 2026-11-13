package sandboxexec

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const servicePath = "/databricks.sandbox.SandboxService/"

type workspaceCredentials struct{ config *config.Config }

func (c workspaceCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.Host, nil)
	if err != nil {
		return nil, err
	}
	if err := c.config.Authenticate(request); err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for key, value := range auth.WorkspaceIDHeaders(c.config) {
		result[strings.ToLower(key)] = value
	}
	for key, values := range request.Header {
		result[strings.ToLower(key)] = strings.Join(values, ",")
	}
	return result, nil
}
func (workspaceCredentials) RequireTransportSecurity() bool { return true }

// Client owns the authenticated command-execution channel for one sandbox.
type Client struct {
	conn *grpc.ClientConn
	name string
}

func New(w *databricks.WorkspaceClient, name string) (*Client, error) {
	host, err := url.Parse(w.Config.Host)
	if err != nil || host.Scheme != "https" || host.Hostname() == "" {
		return nil, errors.New("sandbox command execution requires an HTTPS workspace URL")
	}
	address := host.Host
	if host.Port() == "" {
		address += ":443"
	}
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: w.Config.InsecureSkipVerify})),
		grpc.WithPerRPCCredentials(workspaceCredentials{config: w.Config}))
	if err != nil {
		return nil, fmt.Errorf("create sandbox command channel: %w", err)
	}
	return &Client{conn: conn, name: name}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

// Execute leaves the process running when its output stream is closed. Terminate
// must be called separately to stop the process and its descendants.
func (c *Client) Execute(ctx context.Context, commandID, executable string, args []string) (grpc.ServerStreamingClient[ExecuteCommandResponse], error) {
	ctx = metadata.AppendToOutgoingContext(ctx, "x-databricks-sandbox-name", c.name)
	stream, err := c.conn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true}, servicePath+"ExecuteCommand")
	if err != nil {
		return nil, err
	}
	if err := stream.SendMsg(&ExecuteCommandRequest{Cmd: new(executable), Args: args, Envs: map[string]string{"TZ": "UTC"}, CommandId: new(commandID)}); err != nil {
		return nil, err
	}
	if err := stream.CloseSend(); err != nil {
		return nil, err
	}
	return &grpc.GenericClientStream[ExecuteCommandRequest, ExecuteCommandResponse]{ClientStream: stream}, nil
}

func (c *Client) Terminate(ctx context.Context, commandID string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, "x-databricks-sandbox-name", c.name)
	return c.conn.Invoke(ctx, servicePath+"TerminateCommand", &TerminateCommandRequest{CommandId: new(commandID)}, &TerminateCommandResponse{})
}
