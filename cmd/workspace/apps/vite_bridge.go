package apps

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

const (
	localViteURL    = "http://localhost:5173"
	localViteHMRURL = "ws://localhost:5173/dev-hmr"
	viteHMRProtocol = "vite-hmr"

	// WebSocket timeouts
	wsHandshakeTimeout  = 45 * time.Second
	wsKeepaliveInterval = 20 * time.Second
	wsWriteTimeout      = 5 * time.Second

	// HTTP client timeouts
	httpRequestTimeout  = 60 * time.Second
	httpIdleConnTimeout = 90 * time.Second

	// Bridge operation timeouts
	bridgeFetchTimeout       = 30 * time.Second
	bridgeConnTimeout        = 60 * time.Second
	bridgeTunnelReadyTimeout = 30 * time.Second
)

type ViteBridgeMessage struct {
	Type      string         `json:"type"`
	TunnelID  string         `json:"tunnelId,omitempty"`
	Path      string         `json:"path,omitempty"`
	Method    string         `json:"method,omitempty"`
	Status    int            `json:"status,omitempty"`
	Headers   map[string]any `json:"headers,omitempty"`
	Body      string         `json:"body,omitempty"`
	Viewer    string         `json:"viewer"`
	RequestID string         `json:"requestId"`
	Approved  bool           `json:"approved"`
	Content   string         `json:"content,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// prioritizedMessage represents a message to send through the tunnel websocket
type prioritizedMessage struct {
	messageType int
	data        []byte
	priority    int // 0 = high (HMR), 1 = normal (fetch)
}

type ViteBridge struct {
	ctx                context.Context
	w                  *databricks.WorkspaceClient
	appName            string
	tunnelConn         *websocket.Conn
	hmrConn            *websocket.Conn
	tunnelID           string
	tunnelWriteChan    chan prioritizedMessage
	stopChan           chan struct{}
	stopOnce           sync.Once
	httpClient         *http.Client
	connectionRequests chan *ViteBridgeMessage
}

func NewViteBridge(ctx context.Context, w *databricks.WorkspaceClient, appName string) *ViteBridge {
	// Configure HTTP client optimized for local high-volume requests
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     httpIdleConnTimeout,
		DisableKeepAlives:   false,
		DisableCompression:  false,
	}

	return &ViteBridge{
		ctx:     ctx,
		w:       w,
		appName: appName,
		httpClient: &http.Client{
			Timeout:   httpRequestTimeout,
			Transport: transport,
		},
		stopChan:           make(chan struct{}),
		tunnelWriteChan:    make(chan prioritizedMessage, 100), // Buffered channel for async writes
		connectionRequests: make(chan *ViteBridgeMessage, 10),
	}
}

func (vb *ViteBridge) getAuthHeaders(wsURL string) (http.Header, error) {
	req, err := http.NewRequestWithContext(vb.ctx, "GET", wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	err = vb.w.Config.Authenticate(req)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	return req.Header, nil
}

func (vb *ViteBridge) GetAppDomain() (*url.URL, error) {
	app, err := vb.w.Apps.Get(vb.ctx, apps.GetAppRequest{
		Name: vb.appName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	if app.Url == "" {
		return nil, errors.New("app URL is empty")
	}

	return url.Parse(app.Url)
}

func (vb *ViteBridge) connectToTunnel(appDomain *url.URL) error {
	wsURL := fmt.Sprintf("wss://%s/dev-tunnel", appDomain.Host)

	headers, err := vb.getAuthHeaders(wsURL)
	if err != nil {
		return fmt.Errorf("failed to get auth headers: %w", err)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: wsHandshakeTimeout,
		ReadBufferSize:   256 * 1024, // 256KB read buffer for large assets
		WriteBufferSize:  256 * 1024, // 256KB write buffer for large assets
	}

	conn, resp, err := dialer.Dial(wsURL, headers)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("failed to connect to tunnel (status %d): %w, body: %s", resp.StatusCode, err, string(body))
		}
		return fmt.Errorf("failed to connect to tunnel: %w", err)
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}

	// Configure keepalive to prevent server timeout
	_ = conn.SetReadDeadline(time.Time{})  // No read timeout
	_ = conn.SetWriteDeadline(time.Time{}) // No write timeout

	// Enable pong handler to respond to server pongs (response to our pings)
	conn.SetPongHandler(func(appData string) error {
		log.Debugf(vb.ctx, "[vite_bridge] Received pong from server")
		return nil
	})

	// Enable ping handler to respond to server pings with pongs
	conn.SetPingHandler(func(appData string) error {
		log.Debugf(vb.ctx, "[vite_bridge] Received ping from server, sending pong")
		// Send pong response
		select {
		case vb.tunnelWriteChan <- prioritizedMessage{
			messageType: websocket.PongMessage,
			data:        []byte(appData),
			priority:    0, // High priority
		}:
		case <-time.After(wsWriteTimeout):
			log.Warnf(vb.ctx, "[vite_bridge] Failed to send pong response")
		}
		return nil
	})

	vb.tunnelConn = conn

	// Start keepalive ping goroutine
	go vb.tunnelKeepalive()

	return nil
}

func (vb *ViteBridge) connectToViteHMR() error {
	dialer := websocket.Dialer{
		Subprotocols: []string{viteHMRProtocol},
	}

	conn, resp, err := dialer.Dial(localViteHMRURL, nil)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return fmt.Errorf("failed to connect to Vite HMR: %w", err)
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}

	vb.hmrConn = conn
	log.Infof(vb.ctx, "[vite_bridge] Connected to local Vite HMR WS")
	return nil
}

// tunnelKeepalive sends periodic pings to keep the connection alive
// Remote servers often have 30-60s idle timeouts
func (vb *ViteBridge) tunnelKeepalive() {
	ticker := time.NewTicker(wsKeepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-vb.stopChan:
			return
		case <-ticker.C:
			// Send ping through the write channel to avoid race conditions
			select {
			case vb.tunnelWriteChan <- prioritizedMessage{
				messageType: websocket.PingMessage,
				data:        []byte{},
				priority:    0, // High priority to ensure keepalive
			}:
				log.Debugf(vb.ctx, "[vite_bridge] Sent keepalive ping")
			case <-time.After(wsWriteTimeout):
				log.Warnf(vb.ctx, "[vite_bridge] Failed to send keepalive ping (channel full)")
			}
		}
	}
}

// tunnelWriter handles all writes to the tunnel websocket in a single goroutine
// This eliminates mutex contention and ensures ordered delivery
func (vb *ViteBridge) tunnelWriter(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-vb.stopChan:
			return nil
		case msg := <-vb.tunnelWriteChan:
			if err := vb.tunnelConn.WriteMessage(msg.messageType, msg.data); err != nil {
				log.Errorf(vb.ctx, "[vite_bridge] Failed to write message: %v", err)
				return fmt.Errorf("failed to write to tunnel: %w", err)
			}
		}
	}
}

func (vb *ViteBridge) handleTunnelMessages(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-vb.stopChan:
			return nil
		default:
		}

		_, message, err := vb.tunnelConn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure) {
				log.Infof(vb.ctx, "[vite_bridge] Tunnel closed, reconnecting...")
				time.Sleep(time.Second)

				appDomain, err := vb.GetAppDomain()
				if err != nil {
					return fmt.Errorf("failed to get app domain for reconnection: %w", err)
				}

				if err := vb.connectToTunnel(appDomain); err != nil {
					return fmt.Errorf("failed to reconnect to tunnel: %w", err)
				}
				continue
			}
			return fmt.Errorf("tunnel connection error: %w", err)
		}

		// Debug: Log raw message
		log.Debugf(vb.ctx, "[vite_bridge] Raw message: %s", string(message))

		var msg ViteBridgeMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Errorf(vb.ctx, "[vite_bridge] Failed to parse message: %v", err)
			continue
		}

		// Debug: Log all incoming message types
		log.Debugf(vb.ctx, "[vite_bridge] Received message type: %s", msg.Type)

		if err := vb.handleMessage(&msg); err != nil {
			log.Errorf(vb.ctx, "[vite_bridge] Error handling message: %v", err)
		}
	}
}

func (vb *ViteBridge) handleMessage(msg *ViteBridgeMessage) error {
	switch msg.Type {
	case "tunnel:ready":
		vb.tunnelID = msg.TunnelID
		log.Infof(vb.ctx, "[vite_bridge] Tunnel ID assigned: %s", vb.tunnelID)
		return nil

	case "connection:request":
		vb.connectionRequests <- msg
		return nil

	case "fetch":
		go func(fetchMsg ViteBridgeMessage) {
			if err := vb.handleFetchRequest(&fetchMsg); err != nil {
				log.Errorf(vb.ctx, "[vite_bridge] Error handling fetch request for %s: %v", fetchMsg.Path, err)
			}
		}(*msg)
		return nil

	case "file:read":
		// Handle file read requests in parallel like fetch requests
		go func(fileReadMsg ViteBridgeMessage) {
			if err := vb.handleFileReadRequest(&fileReadMsg); err != nil {
				log.Errorf(vb.ctx, "[vite_bridge] Error handling file read request for %s: %v", fileReadMsg.Path, err)
			}
		}(*msg)
		return nil

	case "hmr:message":
		return vb.handleHMRMessage(msg)

	default:
		log.Warnf(vb.ctx, "[vite_bridge] Unknown message type: %s", msg.Type)
		return nil
	}
}

func (vb *ViteBridge) handleConnectionRequest(msg *ViteBridgeMessage) error {
	cmdio.LogString(vb.ctx, "")
	cmdio.LogString(vb.ctx, "🔔 Connection Request")
	cmdio.LogString(vb.ctx, "   User: "+msg.Viewer)
	cmdio.LogString(vb.ctx, "   Approve this connection? (y/n)")

	// Read from stdin with timeout to prevent indefinite blocking
	inputChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			errChan <- err
			return
		}
		inputChan <- input
	}()

	var approved bool
	select {
	case input := <-inputChan:
		approved = strings.ToLower(strings.TrimSpace(input)) == "y"
	case err := <-errChan:
		return fmt.Errorf("failed to read user input: %w", err)
	case <-time.After(bridgeConnTimeout):
		// Default to denying after timeout
		cmdio.LogString(vb.ctx, "⏱️  Timeout waiting for response, denying connection")
		approved = false
	}

	response := ViteBridgeMessage{
		Type:      "connection:response",
		RequestID: msg.RequestID,
		Viewer:    msg.Viewer,
		Approved:  approved,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal connection response: %w", err)
	}

	// Send through channel instead of direct write
	select {
	case vb.tunnelWriteChan <- prioritizedMessage{
		messageType: websocket.TextMessage,
		data:        responseData,
		priority:    1,
	}:
	case <-time.After(wsWriteTimeout):
		return errors.New("timeout sending connection response")
	}

	if approved {
		cmdio.LogString(vb.ctx, "✅ Approved connection from "+msg.Viewer)
	} else {
		cmdio.LogString(vb.ctx, "❌ Denied connection from "+msg.Viewer)
	}

	return nil
}

func (vb *ViteBridge) handleFetchRequest(msg *ViteBridgeMessage) error {
	targetURL := fmt.Sprintf("%s%s", localViteURL, msg.Path)
	log.Debugf(vb.ctx, "[vite_bridge] Fetch request: %s %s", msg.Method, msg.Path)

	req, err := http.NewRequestWithContext(vb.ctx, msg.Method, targetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := vb.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch from Vite: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	log.Debugf(vb.ctx, "[vite_bridge] Fetch response: %s (status=%d, size=%d bytes)", msg.Path, resp.StatusCode, len(body))

	headers := make(map[string]any, len(resp.Header))
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	metadataResponse := ViteBridgeMessage{
		Type:      "fetch:response:meta",
		Path:      msg.Path,
		Status:    resp.StatusCode,
		Headers:   headers,
		RequestID: msg.RequestID,
	}

	responseData, err := json.Marshal(metadataResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	select {
	case vb.tunnelWriteChan <- prioritizedMessage{
		messageType: websocket.TextMessage,
		data:        responseData,
		priority:    1, // Normal priority
	}:
	case <-time.After(bridgeFetchTimeout):
		return errors.New("timeout sending fetch metadata")
	}

	if len(body) > 0 {
		select {
		case vb.tunnelWriteChan <- prioritizedMessage{
			messageType: websocket.BinaryMessage,
			data:        body,
			priority:    1, // Normal priority
		}:
		case <-time.After(bridgeFetchTimeout):
			return errors.New("timeout sending fetch body")
		}
	}

	return nil
}

const (
	allowedBasePath  = "config/queries"
	allowedExtension = ".sql"
)

func (vb *ViteBridge) handleFileReadRequest(msg *ViteBridgeMessage) error {
	log.Debugf(vb.ctx, "[vite_bridge] File read request: %s", msg.Path)

	if err := validateFilePath(msg.Path); err != nil {
		log.Warnf(vb.ctx, "[vite_bridge] File validation failed for %s: %v", msg.Path, err)
		return vb.sendFileReadError(msg.RequestID, fmt.Sprintf("Invalid file path: %v", err))
	}

	content, err := os.ReadFile(msg.Path)

	response := ViteBridgeMessage{
		Type:      "file:read:response",
		RequestID: msg.RequestID,
	}

	if err != nil {
		log.Errorf(vb.ctx, "[vite_bridge] Failed to read file %s: %v", msg.Path, err)
		response.Error = err.Error()
	} else {
		log.Debugf(vb.ctx, "[vite_bridge] Read file %s (%d bytes)", msg.Path, len(content))
		response.Content = string(content)
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal file read response: %w", err)
	}

	select {
	case vb.tunnelWriteChan <- prioritizedMessage{
		messageType: websocket.TextMessage,
		data:        responseData,
		priority:    1,
	}:
	case <-time.After(wsWriteTimeout):
		return errors.New("timeout sending file read response")
	}

	return nil
}

func validateFilePath(requestedPath string) error {
	// Clean the path to resolve any ../ or ./ components
	cleanPath := filepath.Clean(requestedPath)

	// Get absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Get the working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Construct the allowed base directory (absolute path)
	allowedDir := filepath.Join(cwd, allowedBasePath)

	// Ensure the resolved path is within the allowed directory
	// Add trailing separator to prevent prefix attacks (e.g., queries-malicious/)
	allowedDirWithSep := allowedDir + string(filepath.Separator)
	if absPath != allowedDir && !strings.HasPrefix(absPath, allowedDirWithSep) {
		return fmt.Errorf("path %s is outside allowed directory %s", absPath, allowedBasePath)
	}

	// Ensure the file has the correct extension
	if filepath.Ext(absPath) != allowedExtension {
		return fmt.Errorf("only %s files are allowed, got: %s", allowedExtension, filepath.Ext(absPath))
	}

	// Additional check: no hidden files
	if strings.HasPrefix(filepath.Base(absPath), ".") {
		return errors.New("hidden files are not allowed")
	}

	return nil
}

// Helper to send error response
func (vb *ViteBridge) sendFileReadError(requestID, errorMsg string) error {
	response := ViteBridgeMessage{
		Type:      "file:read:response",
		RequestID: requestID,
		Error:     errorMsg,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal error response: %w", err)
	}

	select {
	case vb.tunnelWriteChan <- prioritizedMessage{
		messageType: websocket.TextMessage,
		data:        responseData,
		priority:    1,
	}:
	case <-time.After(wsWriteTimeout):
		return errors.New("timeout sending file read error")
	}

	return nil
}

func (vb *ViteBridge) handleHMRMessage(msg *ViteBridgeMessage) error {
	log.Debugf(vb.ctx, "[vite_bridge] HMR message received: %s", msg.Body)

	response := ViteBridgeMessage{
		Type: "hmr:client",
		Body: msg.Body,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal HMR message: %w", err)
	}

	// Send HMR with HIGH priority so it doesn't get blocked by fetch requests
	select {
	case vb.tunnelWriteChan <- prioritizedMessage{
		messageType: websocket.TextMessage,
		data:        responseData,
		priority:    0, // HIGH PRIORITY for HMR!
	}:
	case <-time.After(wsWriteTimeout):
		return errors.New("timeout sending HMR message")
	}

	return nil
}

func (vb *ViteBridge) handleViteHMRMessages(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-vb.stopChan:
			return nil
		default:
		}

		_, message, err := vb.hmrConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Infof(vb.ctx, "[vite_bridge] Vite HMR connection closed, reconnecting...")
				time.Sleep(time.Second)
				if err := vb.connectToViteHMR(); err != nil {
					return fmt.Errorf("failed to reconnect to Vite HMR: %w", err)
				}
				continue
			}
			return err
		}

		response := ViteBridgeMessage{
			Type: "hmr:message",
			Body: string(message),
		}

		responseData, err := json.Marshal(response)
		if err != nil {
			log.Errorf(vb.ctx, "[vite_bridge] Failed to marshal Vite HMR message: %v", err)
			continue
		}

		select {
		case vb.tunnelWriteChan <- prioritizedMessage{
			messageType: websocket.TextMessage,
			data:        responseData,
			priority:    0,
		}:
		case <-time.After(wsWriteTimeout):
			log.Errorf(vb.ctx, "[vite_bridge] Timeout sending Vite HMR message")
		}
	}
}

func (vb *ViteBridge) Start() error {
	appDomain, err := vb.GetAppDomain()
	if err != nil {
		return fmt.Errorf("failed to get app domain: %w", err)
	}

	if err := vb.connectToTunnel(appDomain); err != nil {
		return err
	}

	readyChan := make(chan error, 1)
	go func() {
		for vb.tunnelID == "" {
			_, message, err := vb.tunnelConn.ReadMessage()
			if err != nil {
				readyChan <- err
				return
			}

			var msg ViteBridgeMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			if msg.Type == "tunnel:ready" {
				vb.tunnelID = msg.TunnelID
				log.Infof(vb.ctx, "[vite_bridge] Tunnel ID assigned: %s", vb.tunnelID)
				readyChan <- nil
				return
			}
		}
	}()

	select {
	case err := <-readyChan:
		if err != nil {
			return fmt.Errorf("failed waiting for tunnel ready: %w", err)
		}
	case <-time.After(bridgeTunnelReadyTimeout):
		return errors.New("timeout waiting for tunnel ready")
	}

	if err := vb.connectToViteHMR(); err != nil {
		return err
	}

	cmdio.LogString(vb.ctx, fmt.Sprintf("\n🌐 App URL:\n%s?dev=true\n", appDomain.String()))
	cmdio.LogString(vb.ctx, fmt.Sprintf("\n🔗 Shareable URL:\n%s?dev=%s\n", appDomain.String(), vb.tunnelID))

	g, gCtx := errgroup.WithContext(vb.ctx)

	// Start dedicated tunnel writer goroutine
	g.Go(func() error {
		if err := vb.tunnelWriter(gCtx); err != nil {
			return fmt.Errorf("tunnel writer error: %w", err)
		}
		return nil
	})

	// Connection request handler - not in errgroup to avoid blocking other handlers
	go func() {
		for {
			select {
			case msg := <-vb.connectionRequests:
				if err := vb.handleConnectionRequest(msg); err != nil {
					log.Errorf(vb.ctx, "[vite_bridge] Error handling connection request: %v", err)
				}
			case <-gCtx.Done():
				return
			case <-vb.stopChan:
				return
			}
		}
	}()

	g.Go(func() error {
		if err := vb.handleTunnelMessages(gCtx); err != nil {
			return fmt.Errorf("tunnel message handler error: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		if err := vb.handleViteHMRMessages(gCtx); err != nil {
			return fmt.Errorf("vite HMR message handler error: %w", err)
		}
		return nil
	})

	<-gCtx.Done()
	vb.Stop()
	return g.Wait()
}

func (vb *ViteBridge) Stop() {
	vb.stopOnce.Do(func() {
		close(vb.stopChan)

		if vb.tunnelConn != nil {
			_ = vb.tunnelConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			vb.tunnelConn.Close()
		}

		if vb.hmrConn != nil {
			_ = vb.hmrConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			vb.hmrConn.Close()
		}
	})
}
