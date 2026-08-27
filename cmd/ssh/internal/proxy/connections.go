package proxy

import (
	"sync"
	"time"
)

// ConnectionsManager manages concurrent websocket clients and sends a shutdown signal if no
// clients are connected for a specified duration.
type ConnectionsManager struct {
	maxClients      int
	shutdownDelay   time.Duration
	shutdownTimer   *time.Timer
	shutdownTimerMu sync.Mutex
	connections     map[string]*proxyConnection
	connectionsMu   sync.Mutex
	TimedOut        chan bool
}

func NewConnectionsManager(maxClients int, shutdownDelay time.Duration) *ConnectionsManager {
	cm := &ConnectionsManager{
		maxClients:    maxClients,
		shutdownDelay: shutdownDelay,
		connections:   make(map[string]*proxyConnection),
		TimedOut:      make(chan bool),
	}
	cm.startShutdownTimer()
	return cm
}

func (cm *ConnectionsManager) Count() int {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	return len(cm.connections)
}

func (cm *ConnectionsManager) TryAdd(id string, conn *proxyConnection) bool {
	count := cm.Count()
	if count >= cm.maxClients {
		return false
	}
	cm.addConnection(id, conn)
	cm.cancelShutdownTimer()
	return true
}

func (cm *ConnectionsManager) addConnection(id string, conn *proxyConnection) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	cm.connections[id] = conn
}

func (cm *ConnectionsManager) Get(id string) (*proxyConnection, bool) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	conn, exists := cm.connections[id]
	return conn, exists
}

func (cm *ConnectionsManager) Remove(id string) {
	cm.removeConnection(id)
	count := cm.Count()
	if count <= 0 {
		cm.startShutdownTimer()
	}
}

func (cm *ConnectionsManager) removeConnection(id string) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	delete(cm.connections, id)
}

func (cm *ConnectionsManager) startShutdownTimer() {
	cm.shutdownTimerMu.Lock()
	defer cm.shutdownTimerMu.Unlock()
	if cm.shutdownTimer != nil {
		cm.shutdownTimer.Stop()
	}
	cm.shutdownTimer = time.AfterFunc(cm.shutdownDelay, func() {
		cm.TimedOut <- true
	})
}

func (cm *ConnectionsManager) cancelShutdownTimer() {
	cm.shutdownTimerMu.Lock()
	defer cm.shutdownTimerMu.Unlock()
	if cm.shutdownTimer != nil {
		cm.shutdownTimer.Stop()
	}
}
