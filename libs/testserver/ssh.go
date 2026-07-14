package testserver

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// sshTunnelUpgrader upgrades the driver-proxy /ssh request to a websocket.
var sshTunnelUpgrader = websocket.Upgrader{}

// sshTunnelEchoHandler stands in for the tunnel server a real cluster runs behind
// the driver proxy: with no compute locally it just echoes binary frames, enough
// to drive `ssh connect --proxy` end to end. In-process (no `cat` subprocess), so
// it also works on Windows.
func sshTunnelEchoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := sshTunnelUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			// Client closed (its stdin hit EOF), or the connection is gone.
			return
		}
		if messageType != websocket.BinaryMessage {
			continue
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			return
		}
	}
}
