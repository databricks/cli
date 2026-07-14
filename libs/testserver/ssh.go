package testserver

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// sshTunnelUpgrader upgrades the driver-proxy /ssh request to a websocket.
// The zero value accepts the CLI's same-origin-less dialer without extra checks.
var sshTunnelUpgrader = websocket.Upgrader{}

// sshTunnelEchoHandler stands in for the SSH tunnel server that a real cluster
// runs behind the driver proxy. There is no compute locally, so it just echoes
// every binary frame back to the client. That is enough to exercise the client's
// raw byte proxy (`ssh connect --proxy`) end-to-end: the CLI dials the tunnel,
// pipes stdin over the websocket, and the frames it reads back are what a live
// `cat`-style loopback would return. It mirrors the echo server used by the
// proxy package's Go tests, but in-process and without a `cat` subprocess, so it
// also works on Windows.
func sshTunnelEchoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := sshTunnelUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			// A normal closure from the client (its stdin reached EOF) ends the
			// loop; any other read error means the connection is gone anyway.
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
