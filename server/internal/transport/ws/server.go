package ws

import (
	"log"
	"net/http"
	"sealdice-mcsm/internal/app"

	"github.com/gorilla/websocket"
)

// Server manages WebSocket connections.
type Server struct {
	App      *app.Service
	Upgrader websocket.Upgrader
}

// NewServer creates a new WebSocket server.
func NewServer(app *app.Service) *Server {
	return &Server{
		App: app,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for simplicity/plugin access
			},
		},
	}
}

// ServeHTTP handles the WebSocket handshake and connection.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	log.Printf("[WS] New connection from %s", r.RemoteAddr)
	handler := NewHandler(conn, s.App)
	handler.Loop()
}
