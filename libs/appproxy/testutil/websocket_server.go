package testutil

import (
	"bytes"
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
	t      *testing.T
	server *httptest.Server
	conn   *websocket.Conn
}

func (s *WebsocketServer) Start() {
	s.server.Start()
}

func (s *WebsocketServer) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
	if s.server != nil {
		s.server.Close()
	}
}

func (s *WebsocketServer) Addr() string {
	return s.server.Listener.Addr().String()
}

func (s *WebsocketServer) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.t.Log(err)
		return
	}
	s.conn = conn
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			s.t.Log(err)
			return
		}

		err = conn.WriteMessage(websocket.TextMessage, bytes.NewBufferString("Message from client: "+string(message)).Bytes())
		if err != nil {
			s.t.Log(err)
			return
		}
	}
}

// NewWebsocketServer creates a new httptest.Server that will handle websocket connections
// and will echo back the message sent by the client with a prefix "Message from client: "
func NewWebsocketServer(t *testing.T) *WebsocketServer {
	w := &WebsocketServer{t: t}
	w.server = httptest.NewUnstartedServer(http.HandlerFunc(w.wsHandler))
	return w
}
