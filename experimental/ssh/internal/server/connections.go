package server

import (
	"sync"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
)

// connectionsManager manages concurrent websocket clients and sends a shutdown signal if no
// clients are connected for a specified duration.
type connectionsManager struct {
	maxClients      int
	shutdownDelay   time.Duration
	shutdownTimer   *time.Timer
	shutdownTimerMu sync.Mutex
	connections     map[string]*proxy.ProxyConnection
	connectionsMu   sync.Mutex
	TimedOut        chan bool
}

func newConnectionsManager(maxClients int, shutdownDelay time.Duration) *connectionsManager {
	cm := &connectionsManager{
		maxClients:    maxClients,
		shutdownDelay: shutdownDelay,
		connections:   make(map[string]*proxy.ProxyConnection),
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

func (cm *connectionsManager) TryAdd(id string, conn *proxy.ProxyConnection) bool {
	count := cm.Count()
	if count >= cm.maxClients {
		return false
	}
	cm.addConnection(id, conn)
	cm.cancelShutdownTimer()
	return true
}

func (cm *connectionsManager) addConnection(id string, conn *proxy.ProxyConnection) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	cm.connections[id] = conn
}

func (cm *connectionsManager) Get(id string) (*proxy.ProxyConnection, bool) {
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
