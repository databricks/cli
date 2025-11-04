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

func createTestClient(t *testing.T, serverURL string, requestHandoverTick func() <-chan time.Time, errChan chan error) (io.WriteCloser, *testBuffer) {
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
	go func() {
		err := RunClientProxy(ctx, clientInput, clientOutput, requestHandoverTick, createConn)
		if err != nil && !isNormalClosure(err) && !errors.Is(err, context.Canceled) {
			if errChan != nil {
				errChan <- err
			} else {
				t.Errorf("client error: %v", err)
			}
		}
	}()
	return clientInputWriter, clientOutput
}

func TestClientServerEcho(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()
	clientInputWriter, clientOutput := createTestClient(t, server.URL, nil, nil)
	defer clientInputWriter.Close()

	testMsg1 := []byte("test message 1\n")
	_, err := clientInputWriter.Write(testMsg1)
	require.NoError(t, err)
	err = clientOutput.AssertWrite(testMsg1)
	require.NoError(t, err)

	testMsg2 := []byte("test message 2\n")
	_, err = clientInputWriter.Write(testMsg2)
	require.NoError(t, err)
	err = clientOutput.AssertWrite(testMsg2)
	require.NoError(t, err)

	expectedOutput := fmt.Sprintf("%s%s", testMsg1, testMsg2)
	assert.Equal(t, expectedOutput, clientOutput.String())
}

func TestMultipleClients(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()
	clientInputWriter1, clientOutput1 := createTestClient(t, server.URL, nil, nil)
	defer clientInputWriter1.Close()
	clientInputWriter2, clientOutput2 := createTestClient(t, server.URL, nil, nil)
	defer clientInputWriter2.Close()

	messageCount := 10
	expectedClientOutput1 := ""
	expectedClientOutput2 := ""
	for i := range messageCount {
		message := fmt.Appendf(nil, "client 1 message %d\n", i)
		_, err := clientInputWriter1.Write(message)
		require.NoError(t, err)
		err = clientOutput1.AssertWrite(message)
		require.NoError(t, err)
		expectedClientOutput1 += string(message)

		message = fmt.Appendf(nil, "client 2 message %d\n", i)
		_, err = clientInputWriter2.Write(message)
		require.NoError(t, err)
		err = clientOutput2.AssertWrite(message)
		require.NoError(t, err)
		expectedClientOutput2 += string(message)
	}

	assert.Equal(t, expectedClientOutput1, clientOutput1.String())
	assert.Equal(t, expectedClientOutput2, clientOutput2.String())
}

func TestMaxClients(t *testing.T) {
	maxClients := 2
	server := createTestServer(t, maxClients, time.Hour)
	defer server.Close()
	clientInputWriter1, clientOutput1 := createTestClient(t, server.URL, nil, nil)
	defer clientInputWriter1.Close()
	clientInputWriter2, clientOutput2 := createTestClient(t, server.URL, nil, nil)
	defer clientInputWriter2.Close()

	testMsg1 := []byte("test message 1\n")
	_, err := clientInputWriter1.Write(testMsg1)
	require.NoError(t, err)
	err = clientOutput1.AssertWrite(testMsg1)
	require.NoError(t, err)
	_, err = clientInputWriter2.Write(testMsg1)
	require.NoError(t, err)
	err = clientOutput2.AssertWrite(testMsg1)
	require.NoError(t, err)

	errChan := make(chan error, 1)
	clientInputWriter3, _ := createTestClient(t, server.URL, nil, errChan)
	defer clientInputWriter3.Close()
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
	clientInputWriter, clientOutput := createTestClient(t, server.URL, requestHandoverTick, nil)
	defer clientInputWriter.Close()

	expectedOutput := ""

	wg := sync.WaitGroup{}
	wg.Go(func() {
		for i := range TOTAL_MESSAGE_COUNT {
			if i > 0 && i%MESSAGES_PER_CHUNK == 0 {
				handoverChan <- time.Now()
			}
			message := fmt.Appendf(nil, "message %d\n", i)
			_, err := clientInputWriter.Write(message)
			if err != nil {
				t.Errorf("failed to write message %d: %v", i, err)
			}
			expectedOutput += string(message)
		}
	})

	err := clientOutput.WaitForWrite(fmt.Appendf(nil, "message %d\n", TOTAL_MESSAGE_COUNT-1))
	require.NoError(t, err, "failed to receive the last message (%d)", TOTAL_MESSAGE_COUNT-1)

	wg.Wait()

	// clientOutput is created by appending incoming messages as they arrive, so we are also test correct order here
	assert.Equal(t, expectedOutput, clientOutput.String())
}
