package sandboxexec

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

func (c *Client) attach(ctx context.Context, commandID string, sequence int64) (grpc.ServerStreamingClient[ExecuteCommandResponse], error) {
	ctx = metadata.AppendToOutgoingContext(ctx, "x-databricks-sandbox-name", c.name)
	stream, err := c.conn.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true}, servicePath+"AttachCommand")
	if err != nil {
		return nil, err
	}
	if err := stream.SendMsg(&AttachCommandRequest{CommandId: new(commandID), LastSequenceNumber: new(sequence)}); err != nil {
		return nil, err
	}
	if err := stream.CloseSend(); err != nil {
		return nil, err
	}
	// AttachCommandResponse has the same process-event and sequence-number fields.
	return &grpc.GenericClientStream[AttachCommandRequest, ExecuteCommandResponse]{ClientStream: stream}, nil
}

// Output renews command-output streams without restarting the remote process.
type Output struct {
	client     *Client
	ctx        context.Context
	streamCtx  context.Context
	cancel     context.CancelFunc
	stream     grpc.ServerStreamingClient[ExecuteCommandResponse]
	commandID  string
	executable string
	args       []string
	received   bool
	sequence   int64
	lifetime   time.Duration
}

// Follow reattaches from the last sequence number before the route expires.
// Unconfirmed startup retries reuse commandID so the daemon rejects duplicates.
// Cancel ctx to close the output stream.
func (c *Client) Follow(ctx context.Context, commandID, executable string, args []string, lifetime time.Duration) (*Output, error) {
	streamCtx, cancel := context.WithTimeout(ctx, lifetime)
	stream, err := c.Execute(streamCtx, commandID, executable, args)
	if err != nil {
		cancel()
		return nil, err
	}
	return &Output{client: c, ctx: ctx, streamCtx: streamCtx, cancel: cancel, stream: stream, commandID: commandID, executable: executable, args: args, lifetime: lifetime}, nil
}

func (o *Output) Recv() (*ExecuteCommandResponse, error) {
	response, err := o.stream.Recv()
	if err == nil {
		o.sequence = response.GetSequenceNumber()
		o.received = true
		return response, nil
	}
	// A command-output RPC can lose its HTTP/2 transport while the command and
	// SSH WebSocket are still healthy. Reattach instead of restarting the command.
	reattach := o.ctx.Err() == nil && !errors.Is(err, io.EOF) &&
		(errors.Is(o.streamCtx.Err(), context.DeadlineExceeded) || retryableOutputError(err))
	o.cancel()
	if !reattach {
		return nil, err
	}
	response, attachErr := o.reattach()
	if attachErr != nil {
		return nil, errors.Join(fmt.Errorf("reattach command output: %w", attachErr), fmt.Errorf("original command-output error: %w", err))
	}
	return response, nil
}

func (o *Output) reattach() (*ExecuteCommandResponse, error) {
	// Match the WebSocket resume budget for independent command-output outages.
	retryCtx, cancelRetry := context.WithTimeout(o.ctx, time.Minute)
	defer cancelRetry()
	backoff := 250 * time.Millisecond
	execute := false
	var err error
	for {
		if o.ctx.Err() != nil {
			return nil, status.FromContextError(o.ctx.Err()).Err()
		}
		streamCtx, cancel := context.WithTimeout(o.ctx, o.lifetime)
		o.streamCtx = streamCtx //nolint:fatcontext // Each RPC derives from the original session context, not the previous RPC.
		o.cancel = cancel
		// Heartbeats can be 30s apart, so allow the full recovery budget for a response.
		stopRetry := context.AfterFunc(retryCtx, cancel)
		if execute {
			o.stream, err = o.client.Execute(o.streamCtx, o.commandID, o.executable, o.args)
		} else {
			o.stream, err = o.client.attach(o.streamCtx, o.commandID, o.sequence)
		}
		var response *ExecuteCommandResponse
		if err == nil {
			// Streaming RPC failures may surface only on the first receive.
			response, err = o.stream.Recv()
		}
		retryStopped := stopRetry()
		if err == nil && retryStopped {
			o.sequence = response.GetSequenceNumber()
			o.received = true
			return response, nil
		}
		o.cancel()
		if o.ctx.Err() != nil {
			return nil, status.FromContextError(o.ctx.Err()).Err()
		}
		if retryCtx.Err() != nil && (err == nil || status.Code(err) == codes.Canceled) {
			err = status.FromContextError(retryCtx.Err()).Err()
		}
		// A lost initial Execute may not have registered the command. Reusing its
		// ID is safe: the daemon atomically rejects a duplicate before spawning.
		retryStartup := !o.received && ((!execute && status.Code(err) == codes.NotFound) || (execute && status.Code(err) == codes.AlreadyExists))
		if !retryStartup && !retryableOutputError(err) {
			return nil, err
		}
		execute = !o.received && status.Code(err) == codes.NotFound
		timer := time.NewTimer(backoff)
		select {
		case <-retryCtx.Done():
			timer.Stop()
			if o.ctx.Err() != nil {
				return nil, status.FromContextError(o.ctx.Err()).Err()
			}
			return nil, err
		case <-timer.C:
		}
		backoff = min(2*backoff, 2*time.Second)
	}
}

func retryableOutputError(err error) bool {
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Internal:
		return true
	default:
		return false
	}
}
