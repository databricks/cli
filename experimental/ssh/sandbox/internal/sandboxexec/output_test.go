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
