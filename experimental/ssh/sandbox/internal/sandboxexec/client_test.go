package sandboxexec_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/experimental/ssh/sandbox/internal/sandboxexec"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type (
	service      any
	recordedCall struct {
		metadata metadata.MD
		command  string
	}
)

type commandService struct {
	mu    sync.Mutex
	calls []recordedCall
}

func (service *commandService) record(ctx context.Context, id string) {
	md, _ := metadata.FromIncomingContext(ctx)
	service.mu.Lock()
	defer service.mu.Unlock()
	service.calls = append(service.calls, recordedCall{md.Copy(), id})
}

func TestAuthenticatedStreamingExecutionAndTermination(t *testing.T) {
	implementation := &commandService{}
	server := grpc.NewServer()
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: "databricks.sandbox.SandboxService", HandlerType: (*service)(nil),
		Streams: []grpc.StreamDesc{{StreamName: "ExecuteCommand", ServerStreams: true, Handler: func(service any, stream grpc.ServerStream) error {
			request := &sandboxexec.ExecuteCommandRequest{}
			if err := stream.RecvMsg(request); err != nil {
				return err
			}
			implementation.record(stream.Context(), request.GetCommandId())
			if request.Envs["TZ"] != "UTC" {
				return errors.New("expected UTC expiry interpretation")
			}
			if request.GetCmd() != "/remote/databricks" {
				return errors.New("unexpected executable")
			}
			return stream.SendMsg(&sandboxexec.ExecuteCommandResponse{ProcessEvent: &sandboxexec.ProcessEvent{Event: &sandboxexec.ProcessEvent_Start{Start: &sandboxexec.StartEvent{CommandId: request.CommandId}}}})
		}}},
		Methods: []grpc.MethodDesc{{MethodName: "TerminateCommand", Handler: func(service any, ctx context.Context, decode func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
			request := &sandboxexec.TerminateCommandRequest{}
			if err := decode(request); err != nil {
				return nil, err
			}
			implementation.record(ctx, request.GetCommandId())
			return &sandboxexec.TerminateCommandResponse{}, nil
		}}},
	}, implementation)
	defer server.Stop()
	endpoint := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			server.ServeHTTP(w, r)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	endpoint.EnableHTTP2 = true
	endpoint.StartTLS()
	defer endpoint.Close()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: endpoint.URL, Token: "test-token", WorkspaceID: "123", InsecureSkipVerify: true, HTTPTransport: endpoint.Client().Transport})
	require.NoError(t, err)
	client, err := sandboxexec.New(w, "sandboxes/test")
	require.NoError(t, err)
	defer client.Close()
	stream, err := client.Execute(t.Context(), "command-id", "/remote/databricks", []string{"sandbox", "server"})
	require.NoError(t, err)
	event, err := stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, "command-id", event.GetProcessEvent().GetStart().GetCommandId())
	_, err = stream.Recv()
	require.ErrorIs(t, err, io.EOF)
	require.NoError(t, client.Terminate(t.Context(), "command-id"))
	implementation.mu.Lock()
	defer implementation.mu.Unlock()
	require.Len(t, implementation.calls, 2)
	for _, call := range implementation.calls {
		assert.Equal(t, []string{"Bearer test-token"}, call.metadata.Get("authorization"))
		assert.Equal(t, []string{"123"}, call.metadata.Get("x-databricks-workspace-id"))
		assert.Equal(t, []string{"sandboxes/test"}, call.metadata.Get("x-databricks-sandbox-name"))
		assert.Equal(t, "command-id", call.command)
	}
}

func TestClientRejectsPlaintext(t *testing.T) {
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: "http://workspace.test", Token: "test-token"})
	require.NoError(t, err)
	_, err = sandboxexec.New(w, "sandboxes/test")
	require.ErrorContains(t, err, "HTTPS")
}
