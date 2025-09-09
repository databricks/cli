package ssh

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"golang.org/x/sync/errgroup"
)

//go:embed jupyter-init.py
var jupyterInitScript string

// connectionsManager manages concurrent websocket clients and sends a shutdown signal if no
// clients are connected for a specified duration.
type connectionsManager struct {
	maxClients      int
	shutdownDelay   time.Duration
	shutdownTimer   *time.Timer
	shutdownTimerMu sync.Mutex
	connections     map[string]*proxyConnection
	connectionsMu   sync.Mutex
	TimedOut        chan bool
}

func newConnectionsManager(maxClients int, shutdownDelay time.Duration) *connectionsManager {
	cm := &connectionsManager{
		maxClients:    maxClients,
		shutdownDelay: shutdownDelay,
		connections:   make(map[string]*proxyConnection),
		TimedOut:      make(chan bool),
	}
	cm.startShutdownTimer()
	return cm
}

func (cm *connectionsManager) Count() int {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	return len(cm.connections)
}

func (cm *connectionsManager) TryAdd(id string, conn *proxyConnection) bool {
	count := cm.Count()
	if count >= cm.maxClients {
		return false
	}
	cm.addConnection(id, conn)
	cm.cancelShutdownTimer()
	return true
}

func (cm *connectionsManager) addConnection(id string, conn *proxyConnection) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	cm.connections[id] = conn
}

func (cm *connectionsManager) Get(id string) (*proxyConnection, bool) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	conn, exists := cm.connections[id]
	return conn, exists
}

func (cm *connectionsManager) Remove(id string) {
	cm.removeConnection(id)
	count := cm.Count()
	if count <= 0 {
		cm.startShutdownTimer()
	}
}

func (cm *connectionsManager) removeConnection(id string) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	delete(cm.connections, id)
}

func (cm *connectionsManager) startShutdownTimer() {
	cm.shutdownTimerMu.Lock()
	defer cm.shutdownTimerMu.Unlock()
	if cm.shutdownTimer != nil {
		cm.shutdownTimer.Stop()
	}
	cm.shutdownTimer = time.AfterFunc(cm.shutdownDelay, func() {
		cm.TimedOut <- true
	})
}

func (cm *connectionsManager) cancelShutdownTimer() {
	cm.shutdownTimerMu.Lock()
	defer cm.shutdownTimerMu.Unlock()
	if cm.shutdownTimer != nil {
		cm.shutdownTimer.Stop()
	}
}

type sshHandler struct {
	Connections    *connectionsManager
	SSHDConfigPath string
}

func newSSHHandler(connections *connectionsManager, sshDConfigPath string) *sshHandler {
	return &sshHandler{
		Connections:    connections,
		SSHDConfigPath: sshDConfigPath,
	}
}

func (handler *sshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
		return
	}

	if conn, exists := handler.Connections.Get(id); exists && conn != nil {
		log.Info(ctx, fmt.Sprintf("Client with id %s is already connected", id))
		err := conn.AcceptHandover(r.Context(), w, r)
		if err != nil {
			log.Error(ctx, fmt.Sprintf("failed to accept handover: %v", err))
			http.Error(w, "Handover failed", http.StatusInternalServerError)
		} else {
			log.Info(ctx, "Handover accepted for client with id "+id)
		}
		return
	}

	proxy := newProxyConnection(nil)
	err := proxy.Accept(w, r)
	if err != nil {
		log.Error(ctx, fmt.Sprintf("failed to upgrade to websockets: %v", err))
		return
	}
	if handler.Connections.TryAdd(id, proxy) {
		log.Info(ctx, fmt.Sprintf("Client connected, current count: %d", handler.Connections.Count()))
	} else {
		err = proxy.Close()
		if err != nil {
			log.Error(ctx, fmt.Sprintf("failed to close websocket: %v", err))
		}
		return
	}
	defer func() {
		log.Info(ctx, "Closing WebSocket connection")
		handler.Connections.Remove(id)
		log.Info(ctx, fmt.Sprintf("Client disconnected, current count: %d", handler.Connections.Count()))
		err := proxy.Close()
		if err != nil {
			log.Error(ctx, fmt.Sprintf("failed to close websocket: %v", err))
		}
	}()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	g, gCtx := errgroup.WithContext(ctx)

	log.Info(ctx, "New WebSocket connection, starting sshd process")

	sshdCmd := createSSHDProcess(gCtx, handler.SSHDConfigPath)

	sshdCmd.Stderr = os.Stderr

	sshdStdin, err := sshdCmd.StdinPipe()
	if err != nil {
		log.Error(ctx, fmt.Sprintf("failed to get stdin pipe: %v", err))
		return
	}
	defer sshdStdin.Close()

	sshdStdout, err := sshdCmd.StdoutPipe()
	if err != nil {
		log.Error(ctx, fmt.Sprintf("failed to get stdout pipe: %v", err))
		return
	}
	defer sshdStdout.Close()

	g.Go(func() error {
		return sshdCmd.Run()
	})

	g.Go(func() error {
		return proxy.Start(gCtx, sshdStdout, sshdStdin)
	})

	if err := g.Wait(); err != nil {
		log.Error(ctx, fmt.Sprintf("SSHHandler error: %v", err))
	}
}

func handleTimeout(ctx context.Context, timedOutChannel chan bool, shutdownDelay time.Duration) {
	<-timedOutChannel
	log.Info(ctx, fmt.Sprintf("No SSH clients for %v, shutting down...", shutdownDelay))
	os.Exit(0)
}

func findAvailablePort(startPort, maxAttempts int) (int, error) {
	for i := range maxAttempts {
		port := startPort + i
		addr := fmt.Sprintf(":%d", port)

		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found after %d attempts starting from port %d", maxAttempts, startPort)
}

func saveWorkspaceMetadata(ctx context.Context, client *databricks.WorkspaceClient, version, clusterID string, port int) error {
	metadataBytes, err := json.Marshal(PortMetadata{Port: port})
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	contentDir, err := getWorkspaceContentDir(ctx, client, version, clusterID)
	if err != nil {
		return fmt.Errorf("failed to get workspace content directory: %w", err)
	}

	metadataPath := filepath.Join(contentDir, "metadata.json")
	err = os.MkdirAll(filepath.Dir(metadataPath), 0o755)
	if err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", metadataPath, err)
	}

	err = os.WriteFile(metadataPath, metadataBytes, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	log.Info(ctx, fmt.Sprintf("Saved port %d to metadata file: %s", port, metadataPath))
	return nil
}

func saveJupyterInitScript(ctx context.Context) error {
	ipythonStartupDir := os.ExpandEnv("$HOME/.ipython/profile_default/startup")

	err := os.MkdirAll(ipythonStartupDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create IPython startup directory %s: %w", ipythonStartupDir, err)
	}

	initScriptPath := filepath.Join(ipythonStartupDir, "init_script.py")
	err = os.WriteFile(initScriptPath, []byte(jupyterInitScript), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write Jupyter init script to %s: %w", initScriptPath, err)
	}

	log.Info(ctx, "Saved Jupyter init script to: "+initScriptPath)
	return nil
}

type ServerOptions struct {
	Version              string
	MaxClients           int
	ShutdownDelay        time.Duration
	ClusterID            string
	ConfigDir            string
	ServerPrivateKeyName string
	ServerPublicKeyName  string
	DefaultPort          int
	PortRange            int
}

func RunServer(ctx context.Context, client *databricks.WorkspaceClient, opts ServerOptions) error {
	port, err := findAvailablePort(opts.DefaultPort, opts.PortRange)
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Info(ctx, "Starting server on "+listenAddr)

	err = saveWorkspaceMetadata(ctx, client, opts.Version, opts.ClusterID, port)
	if err != nil {
		return fmt.Errorf("failed to save metadata to the workspace: %w", err)
	}

	clientPublicKey := os.Getenv("PUBLIC_SSH_KEY")
	if clientPublicKey == "" {
		return errors.New("PUBLIC_SSH_KEY environment variable is not set")
	}

	sshdConfigPath, err := prepareSSHDConfig(ctx, client, clientPublicKey, opts)
	if err != nil {
		return fmt.Errorf("failed to setup SSH configuration: %w", err)
	}

	err = saveJupyterInitScript(ctx)
	if err != nil {
		return fmt.Errorf("failed to save Jupyter init script: %w", err)
	}

	connections := newConnectionsManager(opts.MaxClients, opts.ShutdownDelay)
	http.Handle("/ssh", newSSHHandler(connections, sshdConfigPath))
	go handleTimeout(ctx, connections.TimedOut, opts.ShutdownDelay)

	http.HandleFunc("/metadata", func(w http.ResponseWriter, r *http.Request) {
		currentUser, err := user.Current()
		if err != nil {
			http.Error(w, "Failed to get current user", http.StatusInternalServerError)
			return
		}
		_, err = io.WriteString(w, currentUser.Username)
		if err != nil {
			http.Error(w, "Failed to write current user", http.StatusInternalServerError)
		}
	})

	return http.ListenAndServe(listenAddr, nil)
}
