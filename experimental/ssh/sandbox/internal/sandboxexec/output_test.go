package sandboxexec_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestOutputReattachesWithoutRestartingCommand(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "rotation", true: "cancellation"}[cancelled], func(t *testing.T) {
			var starts, attaches atomic.Int32
			rpc := grpc.NewServer()
			rpc.RegisterService(&grpc.ServiceDesc{
				ServiceName: "databricks.sandbox.SandboxService", HandlerType: (*service)(nil),
				Streams: []grpc.StreamDesc{
					{StreamName: "ExecuteCommand", ServerStreams: true, Handler: func(_ any, stream grpc.ServerStream) error {
						request := new(sandboxexec.ExecuteCommandRequest)
						if err := stream.RecvMsg(request); err != nil {
							return err
						}
						starts.Add(1)
						if err := stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(7))}); err != nil {
							return err
						}
						<-stream.Context().Done()
						return stream.Context().Err()
					}},
					{StreamName: "AttachCommand", ServerStreams: true, Handler: func(_ any, stream grpc.ServerStream) error {
						request := new(sandboxexec.AttachCommandRequest)
						if err := stream.RecvMsg(request); err != nil {
							return err
						}
						attaches.Add(1)
						require.Equal(t, "command-id", request.GetCommandId())
						require.Equal(t, int64(7), request.GetLastSequenceNumber())
						md, _ := metadata.FromIncomingContext(stream.Context())
						require.Equal(t, []string{"Bearer test-token"}, md.Get("authorization"))
						require.Equal(t, []string{"sandboxes/test"}, md.Get("x-databricks-sandbox-name"))
						return stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(8))})
					}},
				},
			}, &commandService{})
			defer rpc.Stop()
			endpoint := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
					rpc.ServeHTTP(w, r)
					return
				}
				_, _ = io.WriteString(w, `{}`)
			}))
			endpoint.EnableHTTP2 = true
			endpoint.StartTLS()
			defer endpoint.Close()
			workspace, err := databricks.NewWorkspaceClient(&databricks.Config{Host: endpoint.URL, Token: "test-token", InsecureSkipVerify: true, HTTPTransport: endpoint.Client().Transport})
			require.NoError(t, err)
			client, err := sandboxexec.New(workspace, "sandboxes/test")
			require.NoError(t, err)
			defer client.Close()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			output, err := client.Follow(ctx, "command-id", "databricks", nil, 200*time.Millisecond)
			require.NoError(t, err)
			first, err := output.Recv()
			require.NoError(t, err)
			require.Equal(t, int64(7), first.GetSequenceNumber())
			if cancelled {
				cancel()
				_, err := output.Recv()
				require.Equal(t, codes.Canceled, status.Code(err))
				require.Zero(t, attaches.Load())
				return
			}
			second, err := output.Recv()
			require.NoError(t, err)
			require.Equal(t, int64(8), second.GetSequenceNumber())
			_, err = output.Recv()
			require.ErrorIs(t, err, io.EOF)
			require.Equal(t, int32(1), starts.Load())
			require.Equal(t, int32(1), attaches.Load())
		})
	}
}

func newOutputForAttachTest(t *testing.T, sourceFailure codes.Code, attach grpc.StreamHandler) (*sandboxexec.Output, *atomic.Int32, context.CancelFunc) {
	t.Helper()
	var starts atomic.Int32
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
				starts.Add(1)
				assert.Equal(t, "command-id", request.GetCommandId())
				if err := stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(7))}); err != nil {
					return err
				}
				if sourceFailure != codes.OK {
					return status.Error(sourceFailure, "command-output transport interrupted")
				}
				<-stream.Context().Done()
				return stream.Context().Err()
			}},
			{StreamName: "AttachCommand", ServerStreams: true, Handler: attach},
		},
	}, &commandService{})
	endpoint := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			rpc.ServeHTTP(w, r)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	endpoint.EnableHTTP2 = true
	endpoint.StartTLS()
	t.Cleanup(endpoint.Close)
	workspace, err := databricks.NewWorkspaceClient(&databricks.Config{Host: endpoint.URL, Token: "test-token", InsecureSkipVerify: true, HTTPTransport: endpoint.Client().Transport})
	require.NoError(t, err)
	client, err := sandboxexec.New(workspace, "sandboxes/test")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx, cancel := context.WithTimeout(t.Context(), 70*time.Second)
	t.Cleanup(cancel)
	output, err := client.Follow(ctx, "command-id", "databricks", nil, 200*time.Millisecond)
	require.NoError(t, err)
	first, err := output.Recv()
	require.NoError(t, err)
	require.Equal(t, int64(7), first.GetSequenceNumber())
	return output, &starts, cancel
}

func TestOutputRetriesTransientAttachErrors(t *testing.T) {
	for _, code := range []codes.Code{codes.Unavailable, codes.DeadlineExceeded, codes.Internal} {
		t.Run(code.String(), func(t *testing.T) {
			t.Parallel()
			var attaches atomic.Int32
			next := make(chan struct{})
			output, starts, _ := newOutputForAttachTest(t, codes.OK, func(_ any, stream grpc.ServerStream) error {
				request := new(sandboxexec.AttachCommandRequest)
				if err := stream.RecvMsg(request); err != nil {
					return err
				}
				attempt := attaches.Add(1)
				assert.Equal(t, "command-id", request.GetCommandId())
				sequence := int64(7)
				if attempt > 2 {
					sequence = 9
				}
				assert.Equal(t, sequence, request.GetLastSequenceNumber())
				if attempt == 1 || attempt == 3 {
					return status.Error(code, "temporary attach failure")
				}
				if attempt == 2 {
					if err := stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(8))}); err != nil {
						return err
					}
					select {
					case <-next:
					case <-stream.Context().Done():
						return stream.Context().Err()
					}
					if err := stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(9))}); err != nil {
						return err
					}
					<-stream.Context().Done()
					return stream.Context().Err()
				}
				return stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(10))})
			})
			response, err := output.Recv()
			require.NoError(t, err)
			require.Equal(t, int64(8), response.GetSequenceNumber())
			// The successful stream must survive the retry function returning.
			close(next)
			for _, sequence := range []int64{9, 10} {
				response, err = output.Recv()
				require.NoError(t, err)
				require.Equal(t, sequence, response.GetSequenceNumber())
			}
			_, err = output.Recv()
			require.ErrorIs(t, err, io.EOF)
			require.Equal(t, int32(1), starts.Load())
			require.Equal(t, int32(4), attaches.Load())
		})
	}
}

func TestOutputReattachesAfterTransportFailure(t *testing.T) {
	for _, code := range []codes.Code{codes.Unavailable, codes.DeadlineExceeded, codes.Internal} {
		t.Run(code.String(), func(t *testing.T) {
			t.Parallel()
			var attaches atomic.Int32
			output, starts, _ := newOutputForAttachTest(t, code, func(_ any, stream grpc.ServerStream) error {
				request := new(sandboxexec.AttachCommandRequest)
				if err := stream.RecvMsg(request); err != nil {
					return err
				}
				attaches.Add(1)
				assert.Equal(t, "command-id", request.GetCommandId())
				assert.Equal(t, int64(7), request.GetLastSequenceNumber())
				return stream.SendMsg(&sandboxexec.ExecuteCommandResponse{SequenceNumber: new(int64(8))})
			})
			response, err := output.Recv()
			require.NoError(t, err)
			require.Equal(t, int64(8), response.GetSequenceNumber())
			require.Equal(t, int32(1), starts.Load())
			require.Equal(t, int32(1), attaches.Load())
		})
	}
}

func TestOutputDoesNotRetryNonTransientAttachErrors(t *testing.T) {
	for _, code := range []codes.Code{codes.Unauthenticated, codes.PermissionDenied, codes.NotFound, codes.InvalidArgument, codes.Canceled, codes.OK} {
		t.Run(code.String(), func(t *testing.T) {
			t.Parallel()
			var attaches atomic.Int32
			output, starts, _ := newOutputForAttachTest(t, codes.OK, func(_ any, stream grpc.ServerStream) error {
				request := new(sandboxexec.AttachCommandRequest)
				if err := stream.RecvMsg(request); err != nil {
					return err
				}
				attaches.Add(1)
				return status.Error(code, "terminal attach result")
			})
			_, err := output.Recv()
			if code == codes.OK {
				require.ErrorIs(t, err, io.EOF)
			} else {
				require.Equal(t, code, status.Code(err))
			}
			require.Equal(t, int32(1), starts.Load())
			require.Equal(t, int32(1), attaches.Load())
		})
	}
}

func TestOutputCancelsAttachRetry(t *testing.T) {
	for _, phase := range []string{"backoff", "receive"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			var attaches atomic.Int32
			attempted := make(chan struct{})
			output, starts, cancel := newOutputForAttachTest(t, codes.OK, func(_ any, stream grpc.ServerStream) error {
				request := new(sandboxexec.AttachCommandRequest)
				if err := stream.RecvMsg(request); err != nil {
					return err
				}
				if attaches.Add(1) == 1 {
					close(attempted)
				}
				if phase == "receive" {
					<-stream.Context().Done()
					return stream.Context().Err()
				}
				return status.Error(codes.Unavailable, "temporary attach failure")
			})
			result := make(chan error, 1)
			go func() {
				_, err := output.Recv()
				result <- err
			}()
			select {
			case <-attempted:
			case <-time.After(3 * time.Second):
				t.Fatal("attach was not attempted")
			}
			// Let a failed first receive enter its backoff before cancelling.
			select {
			case err := <-result:
				t.Fatalf("receive returned before cancellation: %v", err)
			case <-time.After(50 * time.Millisecond):
			}
			cancel()
			select {
			case err := <-result:
				require.Equal(t, codes.Canceled, status.Code(err))
			case <-time.After(time.Second):
				t.Fatal("attach retry did not stop on cancellation")
			}
			require.Equal(t, int32(1), starts.Load())
			require.Equal(t, int32(1), attaches.Load())
		})
	}
}

func TestOutputAttachRetryExhaustion(t *testing.T) {
	t.Parallel()
	for _, code := range []codes.Code{codes.Unavailable, codes.DeadlineExceeded} {
		t.Run(code.String(), func(t *testing.T) {
			t.Parallel()
			var attaches atomic.Int32
			output, starts, _ := newOutputForAttachTest(t, codes.OK, func(_ any, stream grpc.ServerStream) error {
				request := new(sandboxexec.AttachCommandRequest)
				if err := stream.RecvMsg(request); err != nil {
					return err
				}
				attaches.Add(1)
				assert.Equal(t, "command-id", request.GetCommandId())
				assert.Equal(t, int64(7), request.GetLastSequenceNumber())
				return status.Error(code, "attach remains unavailable")
			})
			started := time.Now()
			result := make(chan error, 1)
			go func() {
				_, err := output.Recv()
				result <- err
			}()
			select {
			case err := <-result:
				require.Equal(t, code, status.Code(err))
			case <-time.After(65 * time.Second):
				t.Fatal("attach retries did not exhaust within their bound")
			}
			require.GreaterOrEqual(t, time.Since(started), time.Minute)
			require.Equal(t, int32(1), starts.Load())
			require.Greater(t, attaches.Load(), int32(5))
		})
	}
}
