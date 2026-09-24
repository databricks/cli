package proxy

import (
	"sync"
	"time"
)

// ConnectionsManager manages concurrent websocket clients and sends a shutdown signal if no
// clients are connected for a specified duration.
type ConnectionsManager struct {
	maxClients    int
	shutdownDelay time.Duration
	shutdownTimer *time.Timer
	connections   map[string]*proxyConnection
	connectionsMu sync.Mutex
	// Methods that touch both mutexes acquire connectionsMu first.
	shutdownTimerMu sync.Mutex
	TimedOut        chan bool
}

func NewConnectionsManager(maxClients int, shutdownDelay time.Duration) *ConnectionsManager {
	cm := &ConnectionsManager{
		maxClients:    maxClients,
		shutdownDelay: shutdownDelay,
		connections:   make(map[string]*proxyConnection),
		TimedOut:      make(chan bool),
	}
	cm.startShutdownTimer(shutdownDelay)
	return cm
}

func (cm *ConnectionsManager) Count() int {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	return len(cm.connections)
}

func (cm *ConnectionsManager) TryAdd(id string, conn *proxyConnection) bool {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	if _, exists := cm.connections[id]; exists {
		return false
	}
	if len(cm.connections) >= cm.maxClients {
		return false
	}
	cm.connections[id] = conn
	cm.cancelShutdownTimer()
	return true
}

func (cm *ConnectionsManager) Get(id string) (*proxyConnection, bool) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	conn, exists := cm.connections[id]
	return conn, exists
}

func (cm *ConnectionsManager) Remove(id string) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	delete(cm.connections, id)
	if len(cm.connections) == 0 {
		cm.startShutdownTimer(cm.shutdownDelay)
	}
}

func (cm *ConnectionsManager) ExtendIdleTimeout(delay time.Duration) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()
	if len(cm.connections) == 0 {
		cm.startShutdownTimer(delay)
	}
}

func (cm *ConnectionsManager) startShutdownTimer(delay time.Duration) {
	cm.shutdownTimerMu.Lock()
	defer cm.shutdownTimerMu.Unlock()
	if cm.shutdownTimer != nil {
		cm.shutdownTimer.Stop()
	}
	cm.shutdownTimer = time.AfterFunc(delay, func() {
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
