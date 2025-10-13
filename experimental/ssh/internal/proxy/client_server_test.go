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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestServer(t *testing.T, maxClients int, shutdownDelay time.Duration) *httptest.Server {
	ctx := t.Context()
	connections := NewConnectionsManager(maxClients, shutdownDelay)
	proxyServer := NewProxyServer(ctx, connections, func(ctx context.Context) *exec.Cmd {
		// 'cat' command reads each line from stdin and sends it to stdout, so we can test end-to-end proxying.
		return exec.CommandContext(ctx, "cat")
	})
	return httptest.NewServer(proxyServer)
}

func createTestClient(t *testing.T, serverURL string, handoverTimeout time.Duration, errChan, createConnChan chan error) (io.WriteCloser, *testBuffer) {
	ctx := t.Context()
	clientInput, clientInputWriter := io.Pipe()
	clientOutput := newTestBuffer(t)
	wsURL := "ws" + serverURL[4:]
	createConn := func(ctx context.Context, connID string) (*websocket.Conn, error) {
		url := fmt.Sprintf("%s?id=%s", wsURL, connID)
		conn, _, err := websocket.DefaultDialer.Dial(url, nil) // nolint:bodyclose
		if createConnChan != nil {
			createConnChan <- err
		}
		return conn, err
	}
	go func() {
		err := RunClientProxy(ctx, clientInput, clientOutput, handoverTimeout, createConn)
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
	clientInputWriter, clientOutput := createTestClient(t, server.URL, time.Hour, nil, nil)
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
	clientInputWriter1, clientOutput1 := createTestClient(t, server.URL, time.Hour, nil, nil)
	defer clientInputWriter1.Close()
	clientInputWriter2, clientOutput2 := createTestClient(t, server.URL, time.Hour, nil, nil)
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
	clientInputWriter1, clientOutput1 := createTestClient(t, server.URL, time.Hour, nil, nil)
	defer clientInputWriter1.Close()
	clientInputWriter2, clientOutput2 := createTestClient(t, server.URL, time.Hour, nil, nil)
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
	clientInputWriter3, _ := createTestClient(t, server.URL, time.Hour, errChan, nil)
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

	maxHandoverCount := 3
	handoverTimeout := 500 * time.Millisecond
	createConnChan := make(chan error, 1)
	clientInputWriter, clientOutput := createTestClient(t, server.URL, handoverTimeout, nil, createConnChan)
	defer clientInputWriter.Close()

	messageCount := 0
	expectedOutput := ""
	sendMessage := func() {
		message := fmt.Appendf(nil, "message %d\n", messageCount)
		_, err := clientInputWriter.Write(message)
		if err != nil {
			t.Errorf("failed to write message %d: %v", messageCount, err)
		}
		messageCount++
		if messageCount > TOTAL_MESSAGE_COUNT {
			t.Errorf("exceeded total message count, test buffer won't work correctly")
		}
		expectedOutput += string(message)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		handoverCount := 0
		for {
			select {
			case <-createConnChan:
				sendMessage()
				handoverCount++
				if handoverCount >= maxHandoverCount {
					return
				}
			default:
				sendMessage()
				time.Sleep(time.Millisecond)
			}
		}
	}()

	wg.Wait()

	for i := 0; i < messageCount; {
		// Client can receive multiple echo messages in one response,
		// so we split them again and verify each one.
		data, err := clientOutput.WaitForWrite()
		require.NoError(t, err, "failed to receive message %d", i)
		lines := strings.SplitSeq(string(data), "\n")
		for line := range lines {
			if line != "" {
				assert.Equal(t, fmt.Sprintf("message %d\n", i), line+"\n")
				i++
			}
		}
	}

	assert.Equal(t, expectedOutput, clientOutput.String())
}
