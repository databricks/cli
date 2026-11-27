package sandboxexec_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/sandbox/internal/sandboxexec"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

func newCommandTestClient(t *testing.T, server *grpc.Server) *sandboxexec.Client {
	t.Helper()
	endpoint := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			server.ServeHTTP(w, r)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	endpoint.EnableHTTP2 = true
	endpoint.StartTLS()
	t.Cleanup(endpoint.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: endpoint.URL, Token: "test-token", WorkspaceID: "123", InsecureSkipVerify: true, HTTPTransport: endpoint.Client().Transport})
	require.NoError(t, err)
	client, err := sandboxexec.New(w, "sandboxes/test")
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, client.Close()) })
	return client
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
	client := newCommandTestClient(t, server)
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

func TestOutputRecoversFromLongOutputOnlyOutage(t *testing.T) {
	t.Parallel()
	const outage = 5 * time.Second
	started := sync.OnceValue(time.Now)
	var attaches atomic.Int32
	output, starts, _ := newOutputForAttachTest(t, codes.Unavailable, func(_ any, stream grpc.ServerStream) error {
		request := new(sandboxexec.AttachCommandRequest)
		if err := stream.RecvMsg(request); err != nil {
			return err
		}
		attaches.Add(1)
		assert.Equal(t, "command-id", request.GetCommandId())
		assert.Equal(t, int64(7), request.GetLastSequenceNumber())
		if time.Since(started()) < outage {
			return status.Error(codes.Unavailable, "command output temporarily unavailable")
		}
		return stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(8))})
	})
	response, err := output.Recv()
	require.NoError(t, err)
	assert.Equal(t, int64(8), response.GetSequenceNumber())
	assert.GreaterOrEqual(t, time.Since(started()), outage)
	assert.Greater(t, attaches.Load(), int32(5))
	_, err = output.Recv()
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, int32(1), starts.Load())
}

func TestOutputReattachTiming(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"stalled", "recovered"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			var starts, attaches atomic.Int32
			attemptDurations := make(chan time.Duration, 8)
			server := grpc.NewServer()
			t.Cleanup(server.Stop)
			server.RegisterService(&grpc.ServiceDesc{
				ServiceName: "databricks.sandbox.SandboxService", HandlerType: (*service)(nil),
				Streams: []grpc.StreamDesc{
					{StreamName: "ExecuteCommand", ServerStreams: true, Handler: func(_ any, stream grpc.ServerStream) error {
						request := new(sandboxexec.ExecuteCommandRequest)
						if err := stream.RecvMsg(request); err != nil {
							return err
						}
						starts.Add(1)
						assert.Equal(t, "command-id", request.GetCommandId())
						if err := stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(7))}); err != nil {
							return err
						}
						return status.Error(codes.Unavailable, "command-output transport interrupted")
					}},
					{StreamName: "AttachCommand", ServerStreams: true, Handler: func(_ any, stream grpc.ServerStream) error {
						request := new(sandboxexec.AttachCommandRequest)
						if err := stream.RecvMsg(request); err != nil {
							return err
						}
						attaches.Add(1)
						assert.Equal(t, "command-id", request.GetCommandId())
						assert.Equal(t, int64(7), request.GetLastSequenceNumber())
						if scenario == "stalled" {
							started := time.Now()
							<-stream.Context().Done()
							attemptDurations <- time.Since(started)
							return stream.Context().Err()
						}
						// Match the production heartbeat delay before the first output.
						timer := time.NewTimer(30 * time.Second)
						defer timer.Stop()
						select {
						case <-stream.Context().Done():
							return stream.Context().Err()
						case <-timer.C:
						}
						if err := stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(8))}); err != nil {
							return err
						}
						// The recovered stream must also outlive the recovery deadline.
						timer.Reset(31 * time.Second)
						select {
						case <-stream.Context().Done():
							return stream.Context().Err()
						case <-timer.C:
						}
						return stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(9))})
					}},
				},
			}, &commandService{})
			client := newCommandTestClient(t, server)
			ctx, cancel := context.WithTimeout(t.Context(), 70*time.Second)
			defer cancel()
			output, err := client.Follow(ctx, "command-id", "databricks", nil, 2*time.Minute)
			require.NoError(t, err)
			response, err := output.Recv()
			require.NoError(t, err)
			require.Equal(t, int64(7), response.GetSequenceNumber())
			started := time.Now()
			response, err = output.Recv()
			if scenario == "recovered" {
				require.NoError(t, err)
				require.Equal(t, int64(8), response.GetSequenceNumber())
				assert.GreaterOrEqual(t, time.Since(started), 30*time.Second)
				response, err = output.Recv()
				require.NoError(t, err)
				assert.Equal(t, int64(9), response.GetSequenceNumber())
				_, err = output.Recv()
				require.ErrorIs(t, err, io.EOF)
			} else {
				require.Equal(t, codes.DeadlineExceeded, status.Code(err))
				select {
				case elapsed := <-attemptDurations:
					assert.InDelta(t, float64(time.Minute), float64(elapsed), float64(2*time.Second))
				case <-time.After(time.Second):
					t.Fatal("stalled attach was not canceled")
				}
			}
			elapsed := time.Since(started)
			assert.GreaterOrEqual(t, elapsed, time.Minute)
			assert.Less(t, elapsed, 65*time.Second)
			assert.Equal(t, int32(1), starts.Load())
			assert.Equal(t, int32(1), attaches.Load())
		})
	}
}

func TestOutputRecoversUnconfirmedExecution(t *testing.T) {
	for _, registered := range []bool{false, true} {
		t.Run(fmt.Sprintf("registered=%t", registered), func(t *testing.T) {
			t.Parallel()
			var executes, attaches, processes atomic.Int32
			rpc := grpc.NewServer()
			t.Cleanup(rpc.Stop)
			rpc.RegisterService(&grpc.ServiceDesc{
				ServiceName: "databricks.sandbox.SandboxService", HandlerType: (*service)(nil),
				Streams: []grpc.StreamDesc{
					{StreamName: "ExecuteCommand", ServerStreams: true, Handler: func(_ any, stream grpc.ServerStream) error {
						request := new(sandboxexec.ExecuteCommandRequest)
						if err := stream.RecvMsg(request); err != nil {
							return err
						}
						assert.Equal(t, "command-id", request.GetCommandId())
						assert.Equal(t, "python3", request.GetCmd())
						assert.Equal(t, []string{"-c", "server"}, request.GetArgs())
						attempt := executes.Add(1)
						if attempt == 1 {
							if registered {
								processes.Add(1)
							}
							return status.Error(codes.Unavailable, "execute transport interrupted")
						}
						if registered {
							return status.Error(codes.AlreadyExists, "command_id is already in use")
						}
						processes.Add(1)
						return stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(1))})
					}},
					{StreamName: "AttachCommand", ServerStreams: true, Handler: func(_ any, stream grpc.ServerStream) error {
						request := new(sandboxexec.AttachCommandRequest)
						if err := stream.RecvMsg(request); err != nil {
							return err
						}
						assert.Equal(t, "command-id", request.GetCommandId())
						assert.Zero(t, request.GetLastSequenceNumber())
						if attaches.Add(1) == 1 {
							return status.Error(codes.NotFound, "command not found")
						}
						return stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(1))})
					}},
				},
			}, &commandService{})
			client := newCommandTestClient(t, rpc)
			output, err := client.Follow(t.Context(), "command-id", "python3", []string{"-c", "server"}, time.Minute)
			require.NoError(t, err)
			response, err := output.Recv()
			require.NoError(t, err)
			assert.Equal(t, int64(1), response.GetSequenceNumber())
			assert.EqualValues(t, 2, executes.Load())
			assert.EqualValues(t, 1, processes.Load())
			expectedAttaches := int32(1)
			if registered {
				expectedAttaches = 2
			}
			assert.Equal(t, expectedAttaches, attaches.Load())
			_, err = output.Recv()
			assert.ErrorIs(t, err, io.EOF)
		})
	}
}

func TestOutputDoesNotRestartConfirmedMissingCommand(t *testing.T) {
	var attaches atomic.Int32
	output, starts, _ := newOutputForAttachTest(t, codes.Unavailable, func(_ any, stream grpc.ServerStream) error {
		request := new(sandboxexec.AttachCommandRequest)
		if err := stream.RecvMsg(request); err != nil {
			return err
		}
		attaches.Add(1)
		return status.Error(codes.NotFound, "confirmed command no longer exists")
	})
	_, err := output.Recv()
	require.Equal(t, codes.NotFound, status.Code(err))
	assert.ErrorContains(t, err, "original command-output error")
	assert.EqualValues(t, 1, starts.Load())
	assert.EqualValues(t, 1, attaches.Load())
}
