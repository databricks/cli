package testserver

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebsocketServer struct {
	server *httptest.Server
	conn   *websocket.Conn
}

func (s *WebsocketServer) Start() {
	s.server.Start()
}

func (s *WebsocketServer) Close() {
	s.conn.Close()
	s.server.Close()
}

func (s *WebsocketServer) Addr() string {
	return s.server.Listener.Addr().String()
}

func (s *WebsocketServer) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	s.conn = conn
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		err = conn.WriteMessage(websocket.TextMessage, bytes.NewBufferString("Message from client: "+string(message)).Bytes())
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

// NewWebsocketServer creates a new httptest.Server that will handle websocket connections
// and will echo back the message sent by the client with a prefix "Message from client: "
func NewWebsocketServer(t *testing.T) *WebsocketServer {
	w := &WebsocketServer{}
	w.server = httptest.NewUnstartedServer(http.HandlerFunc(w.wsHandler))
	return w
}
