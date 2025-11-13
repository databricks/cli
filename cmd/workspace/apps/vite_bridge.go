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
)

const (
	localViteURL    = "http://localhost:5173"
	localViteHMRURL = "ws://localhost:5173/dev-hmr"
	viteHMRProtocol = "vite-hmr"
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

type ViteBridge struct {
	ctx        context.Context
	w          *databricks.WorkspaceClient
	appName    string
	tunnelConn *websocket.Conn
	hmrConn    *websocket.Conn
	tunnelID   string
	mu         sync.Mutex
	stopChan   chan struct{}
	stopOnce   sync.Once
	httpClient *http.Client
}

func NewViteBridge(ctx context.Context, w *databricks.WorkspaceClient, appName string) *ViteBridge {
	// Configure HTTP client optimized for local high-volume requests
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  false,
	}

	return &ViteBridge{
		ctx:     ctx,
		w:       w,
		appName: appName,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		stopChan: make(chan struct{}),
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
		HandshakeTimeout: 45 * time.Second,
		ReadBufferSize:   32 * 1024, // 32KB read buffer
		WriteBufferSize:  32 * 1024, // 32KB write buffer for large assets
	}

	conn, resp, err := dialer.Dial(wsURL, headers)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to connect to tunnel (status %d): %w, body: %s", resp.StatusCode, err, string(body))
		}
		return fmt.Errorf("failed to connect to tunnel: %w", err)
	}

	vb.tunnelConn = conn
	return nil
}

func (vb *ViteBridge) connectToViteHMR() error {
	dialer := websocket.Dialer{
		Subprotocols: []string{viteHMRProtocol},
	}

	conn, _, err := dialer.Dial(localViteHMRURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Vite HMR: %w", err)
	}

	vb.hmrConn = conn
	log.Infof(vb.ctx, "[vite_bridge] Connected to local Vite HMR WS")
	return nil
}

func (vb *ViteBridge) handleTunnelMessages() error {
	for {
		select {
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
		return vb.handleConnectionRequest(msg)

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

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	approved := strings.ToLower(strings.TrimSpace(input)) == "y"

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

	vb.mu.Lock()
	defer vb.mu.Unlock()

	if err := vb.tunnelConn.WriteMessage(websocket.TextMessage, responseData); err != nil {
		return fmt.Errorf("failed to send connection response: %w", err)
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

	vb.mu.Lock()

	if err := vb.tunnelConn.WriteMessage(websocket.TextMessage, responseData); err != nil {
		vb.mu.Unlock()
		return fmt.Errorf("failed to send metadata: %w", err)
	}

	if len(body) > 0 {
		if err := vb.tunnelConn.WriteMessage(websocket.BinaryMessage, body); err != nil {
			vb.mu.Unlock()
			return fmt.Errorf("failed to send binary body: %w", err)
		}
	}

	vb.mu.Unlock()
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

	vb.mu.Lock()
	defer vb.mu.Unlock()

	return vb.tunnelConn.WriteMessage(websocket.TextMessage, responseData)
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
	if !strings.HasPrefix(absPath, allowedDir) {
		return fmt.Errorf("path %s is outside allowed directory %s", absPath, allowedBasePath)
	}

	// Ensure the file has the correct extension
	if filepath.Ext(absPath) != allowedExtension {
		return fmt.Errorf("only %s files are allowed, got: %s", allowedExtension, filepath.Ext(absPath))
	}

	// Additional check: no hidden files
	if strings.HasPrefix(filepath.Base(absPath), ".") {
		return fmt.Errorf("hidden files are not allowed")
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

	vb.mu.Lock()
	defer vb.mu.Unlock()

	return vb.tunnelConn.WriteMessage(websocket.TextMessage, responseData)
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

	vb.mu.Lock()
	defer vb.mu.Unlock()
	return vb.tunnelConn.WriteMessage(websocket.TextMessage, responseData)
}

func (vb *ViteBridge) handleViteHMRMessages() error {
	for {
		select {
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

		vb.mu.Lock()
		err = vb.tunnelConn.WriteMessage(websocket.TextMessage, responseData)
		vb.mu.Unlock()

		if err != nil {
			log.Errorf(vb.ctx, "[vite_bridge] Failed to send Vite HMR message to tunnel: %v", err)
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
	case <-time.After(30 * time.Second):
		return errors.New("timeout waiting for tunnel ready")
	}

	if err := vb.connectToViteHMR(); err != nil {
		return err
	}

	cmdio.LogString(vb.ctx, fmt.Sprintf("\n🌐 App URL:\n%s?dev=true\n", appDomain.String()))
	cmdio.LogString(vb.ctx, fmt.Sprintf("\n🔗 Shareable URL:\n%s?dev=%s\n", appDomain.String(), vb.tunnelID))

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := vb.handleTunnelMessages(); err != nil {
			errChan <- fmt.Errorf("tunnel message handler error: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := vb.handleViteHMRMessages(); err != nil {
			errChan <- fmt.Errorf("Vite HMR message handler error: %w", err)
		}
	}()

	select {
	case err := <-errChan:
		vb.Stop()
		wg.Wait()
		return err
	case <-vb.ctx.Done():
		vb.Stop()
		wg.Wait()
		return vb.ctx.Err()
	}
}

func (vb *ViteBridge) Stop() {
	vb.stopOnce.Do(func() {
		close(vb.stopChan)

		if vb.tunnelConn != nil {
			vb.tunnelConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			vb.tunnelConn.Close()
		}

		if vb.hmrConn != nil {
			vb.hmrConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			vb.hmrConn.Close()
		}
	})
}
