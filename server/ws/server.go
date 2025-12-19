package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// AppInterface defines the methods required from the Application layer.
// This helps to decouple ws package from app package, avoiding cyclic dependencies if any.
// But since ws imports app structs in original code (indirectly), we can import app interface.
// Ideally, `app` should import `ws` for `MessageSender`, so `ws` should NOT import `app`.
// The user structure has `ws` and `app`.
// `app` imports `ws` (for MessageSender).
// So `ws` CANNOT import `app` directly without interface or cyclic dependency.
// We define an interface here for the App logic we need.

type Application interface {
	Bind(alias, instanceID string) error
	Control(target, action string) error
	Status(target string) (interface{}, error)
	Relogin(sender MessageSender, params map[string]interface{}) error
	ContinueRelogin() error
	CancelRelogin() error
}

// Server manages WebSocket connections.
type Server struct {
	App      Application
	Upgrader websocket.Upgrader
}

// NewServer creates a new WebSocket server.
func NewServer(app Application) *Server {
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
	// Auth check
	if err := ValidateToken(r); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	log.Printf("[WS] New connection from %s", r.RemoteAddr)
	
	sender := &Sender{Conn: conn}
	s.handleConnection(conn, sender)
}

func (s *Server) handleConnection(conn *websocket.Conn, sender *Sender) {
	defer func() {
		s.App.CancelRelogin() // Cancel any ongoing FSM if connection drops
		conn.Close()
	}()

	for {
		var req Request
		if err := conn.ReadJSON(&req); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Read error: %v", err)
			}
			break
		}
		
		s.dispatchRequest(req, sender)
	}
}

func (s *Server) dispatchRequest(req Request, sender *Sender) {
	var resp Response
	resp.RequestID = req.RequestID

	var err error
	var data interface{}

	switch req.Command {
	case "bind":
		alias, _ := req.Params["alias"].(string)
		instanceID, _ := req.Params["instance_id"].(string)
		if alias == "" || instanceID == "" {
			err = logAndError("missing alias or instance_id")
		} else {
			err = s.App.Bind(alias, instanceID)
			data = "bound"
		}

	case "start", "stop", "restart", "fstop":
		target, _ := req.Params["target"].(string)
		if target == "" {
			err = logAndError("missing target")
		} else {
			err = s.App.Control(target, req.Command)
			data = "ok"
		}

	case "status":
		target, _ := req.Params["target"].(string)
		data, err = s.App.Status(target)

	case "relogin":
		// This triggers async FSM. We return "accepted" immediately.
		err = s.App.Relogin(sender, req.Params)
		if err == nil {
			data = "relogin started"
		}

	case "continue":
		// User confirms waiting relogin
		err = s.App.ContinueRelogin()
		data = "signal sent"

	default:
		err = logAndError("unknown command: " + req.Command)
	}

	if err != nil {
		resp.Code = -1
		resp.Message = err.Error()
	} else {
		resp.Code = 0
		resp.Message = "success"
		resp.Data = data
	}

	// Send immediate response
	sender.Send(resp)
}
