//go:build !windows

// TODO: figure out what command can we use on Windows for the echo server
package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestServer(t *testing.T, maxClients int, shutdownDelay time.Duration) *httptest.Server {
	ctx := cmdio.MockDiscard(t.Context())
	connections := NewConnectionsManager(maxClients, shutdownDelay)
	proxyServer := NewProxyServer(ctx, connections, func(ctx context.Context) *exec.Cmd {
		// 'cat' command reads each line from stdin and sends it to stdout, so we can test end-to-end proxying.
		// '-u' option is used to disable output buffering.
		return exec.CommandContext(ctx, "cat", "-u")
	})
	return httptest.NewServer(proxyServer)
}

type testClient struct {
	InputWriter io.WriteCloser
	Output      *testBuffer
	Cleanup     func()
}

func createTestClient(t *testing.T, serverURL string, requestHandoverTick func() <-chan time.Time, errChan chan error) *testClient {
	ctx := cmdio.MockDiscard(t.Context())
	clientInput, clientInputWriter := io.Pipe()
	clientOutput := newTestBuffer(t)
	wsURL := "ws" + serverURL[4:]
	createConn := func(ctx context.Context, connID string) (*websocket.Conn, error) {
		url := fmt.Sprintf("%s?id=%s", wsURL, connID)
		conn, _, err := websocket.DefaultDialer.Dial(url, nil) // nolint:bodyclose
		return conn, err
	}
	if requestHandoverTick == nil {
		requestHandoverTick = func() <-chan time.Time {
			return time.After(time.Hour)
		}
	}
	wg := sync.WaitGroup{}
	wg.Go(func() {
		err := RunClientProxy(ctx, clientInput, clientOutput, requestHandoverTick, createConn)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, io.ErrClosedPipe) {
			if errChan != nil {
				errChan <- err
			} else {
				t.Errorf("client error: %v", err)
			}
		}
	})
	return &testClient{
		InputWriter: clientInputWriter,
		Output:      clientOutput,
		Cleanup: func() {
			clientInput.Close()
			clientInputWriter.Close()
			wg.Wait()
		},
	}
}

func TestClientServerEcho(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()
	client := createTestClient(t, server.URL, nil, nil)
	defer client.Cleanup()

	testMsg1 := []byte("test message 1\n")
	_, err := client.InputWriter.Write(testMsg1)
	require.NoError(t, err)
	err = client.Output.AssertWrite(testMsg1)
	require.NoError(t, err)

	testMsg2 := []byte("test message 2\n")
	_, err = client.InputWriter.Write(testMsg2)
	require.NoError(t, err)
	err = client.Output.AssertWrite(testMsg2)
	require.NoError(t, err)

	expectedOutput := fmt.Sprintf("%s%s", testMsg1, testMsg2)
	assert.Equal(t, expectedOutput, client.Output.String())
}

func TestMultipleClients(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()
	client1 := createTestClient(t, server.URL, nil, nil)
	defer client1.Cleanup()
	client2 := createTestClient(t, server.URL, nil, nil)
	defer client2.Cleanup()

	messageCount := 10
	expectedClientOutput1 := ""
	expectedClientOutput2 := ""
	for i := range messageCount {
		message := fmt.Appendf(nil, "client 1 message %d\n", i)
		_, err := client1.InputWriter.Write(message)
		require.NoError(t, err)
		err = client1.Output.AssertWrite(message)
		require.NoError(t, err)
		expectedClientOutput1 += string(message)

		message = fmt.Appendf(nil, "client 2 message %d\n", i)
		_, err = client2.InputWriter.Write(message)
		require.NoError(t, err)
		err = client2.Output.AssertWrite(message)
		require.NoError(t, err)
		expectedClientOutput2 += string(message)
	}

	assert.Equal(t, expectedClientOutput1, client1.Output.String())
	assert.Equal(t, expectedClientOutput2, client2.Output.String())
}

func TestMaxClients(t *testing.T) {
	maxClients := 2
	server := createTestServer(t, maxClients, time.Hour)
	defer server.Close()
	client1 := createTestClient(t, server.URL, nil, nil)
	defer client1.Cleanup()
	client2 := createTestClient(t, server.URL, nil, nil)
	defer client2.Cleanup()

	testMsg1 := []byte("test message 1\n")
	_, err := client1.InputWriter.Write(testMsg1)
	require.NoError(t, err)
	err = client1.Output.AssertWrite(testMsg1)
	require.NoError(t, err)
	_, err = client2.InputWriter.Write(testMsg1)
	require.NoError(t, err)
	err = client2.Output.AssertWrite(testMsg1)
	require.NoError(t, err)

	errChan := make(chan error, 1)
	client3 := createTestClient(t, server.URL, nil, errChan)
	defer client3.Cleanup()
	select {
	case err = <-errChan:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("expected error due to max clients reached, but got none")
	}
}

func TestHandover(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	handoverChan := make(chan time.Time)
	requestHandoverTick := func() <-chan time.Time {
		return handoverChan
	}
	client := createTestClient(t, server.URL, requestHandoverTick, nil)
	defer client.Cleanup()

	expectedOutput := ""

	wg := sync.WaitGroup{}
	wg.Go(func() {
		for i := range TOTAL_MESSAGE_COUNT {
			if i > 0 && i%MESSAGES_PER_CHUNK == 0 && i < TOTAL_MESSAGE_COUNT-1 {
				handoverChan <- time.Now()
			}
			message := fmt.Appendf(nil, "message %d\n", i)
			_, err := client.InputWriter.Write(message)
			if err != nil {
				t.Errorf("failed to write message %d: %v", i, err)
			}
			expectedOutput += string(message)
		}
	})

	err := client.Output.WaitForWrite(fmt.Appendf(nil, "message %d\n", TOTAL_MESSAGE_COUNT-1))
	require.NoError(t, err, "failed to receive the last message (%d)", TOTAL_MESSAGE_COUNT-1)

	wg.Wait()

	// client.Output is created by appending incoming messages as they arrive, so we are also test correct order here
	assert.Equal(t, expectedOutput, client.Output.String())
}

// Tests handovers in quick succession with few messages in between.
// Not a real world scenario, but it can help uncover potential race conditions or deadlocks.
func TestQuickHandover(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	handoverChan := make(chan time.Time)
	requestHandoverTick := func() <-chan time.Time {
		return handoverChan
	}
	client := createTestClient(t, server.URL, requestHandoverTick, nil)
	defer client.Cleanup()

	expectedOutput := ""

	wg := sync.WaitGroup{}
	wg.Go(func() {
		for i := range 16 {
			if i == 4 || i == 8 || i == 12 {
				handoverChan <- time.Now()
			}
			message := fmt.Appendf(nil, "message %d\n", i)
			_, err := client.InputWriter.Write(message)
			if err != nil {
				t.Errorf("failed to write message %d: %v", i, err)
			}
			expectedOutput += string(message)
		}
	})

	err := client.Output.WaitForWrite(fmt.Appendf(nil, "message %d\n", 15))
	require.NoError(t, err, "failed to receive the last message (%d)", 15)

	wg.Wait()

	assert.Equal(t, expectedOutput, client.Output.String())
}
