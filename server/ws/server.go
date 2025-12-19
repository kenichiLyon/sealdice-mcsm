package ws

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"sealdice-mcsm/server/app"
)

type App interface {
	Bind(ctx context.Context, alias, instanceID string) error
	Control(ctx context.Context, target string, action app.ControlAction) error
	Status(ctx context.Context, target string) (any, error)
	Relogin(ctx context.Context, sender app.Sender, requestID string, alias string) error
	ContinueRelogin(ctx context.Context, alias string)
	CancelRelogin(ctx context.Context, alias string)
}

type Sender interface {
	SendResponse(requestID string, code int, data any)
	SendEvent(name string, data any)
}

type Server struct {
	app App
	up  websocket.Upgrader
}

func NewServer(a App) *Server {
	return &Server{
		app: a,
		up: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.up.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer conn.Close()
	s.handleConn(conn)
}

type sender struct {
	conn *websocket.Conn
}

func (s *sender) SendResponse(requestID string, code int, data any) {
	_ = s.conn.WriteJSON(Response{
		Type:      "response",
		RequestID: requestID,
		Code:      code,
		Data:      data,
	})
}

func (s *sender) SendEvent(name string, data any) {
	_ = s.conn.WriteJSON(Event{
		Type:  "event",
		Event: name,
		Data:  data,
	})
}

func (s *Server) handleConn(conn *websocket.Conn) {
	sndr := &sender{conn: conn}
	for {
		_, b, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var req Request
		if err := json.Unmarshal(b, &req); err != nil {
			_ = conn.WriteJSON(Response{Type: "response", Code: 400, Message: "bad request"})
			continue
		}
		ctx := context.Background()
		switch req.Command {
		case "bind":
			alias := req.Params["alias"]
			instance := req.Params["instance_id"]
			if err := s.app.Bind(ctx, alias, instance); err != nil {
				sndr.SendResponse(req.RequestID, 500, map[string]string{"error": err.Error()})
			} else {
				sndr.SendResponse(req.RequestID, 200, map[string]string{"status": "ok"})
			}
		case "start", "stop", "restart", "fstop":
			target := req.Params["target"]
			action := app.ControlAction(req.Command)
			if err := s.app.Control(ctx, target, action); err != nil {
				sndr.SendResponse(req.RequestID, 500, map[string]string{"error": err.Error()})
			} else {
				sndr.SendResponse(req.RequestID, 200, map[string]string{"status": "ok"})
			}
		case "status":
			target := req.Params["target"]
			data, err := s.app.Status(ctx, target)
			if err != nil {
				sndr.SendResponse(req.RequestID, 500, map[string]string{"error": err.Error()})
			} else {
				sndr.SendResponse(req.RequestID, 200, data)
			}
		case "relogin":
			alias := req.Params["target"]
			if alias == "" {
				sndr.SendResponse(req.RequestID, 400, map[string]string{"error": "target required"})
				continue
			}
			if err := s.app.Relogin(ctx, sndr, req.RequestID, alias); err != nil {
				sndr.SendResponse(req.RequestID, 409, map[string]string{"error": err.Error()})
			}
		case "continue":
			alias := req.Params["target"]
			s.app.ContinueRelogin(ctx, alias)
			sndr.SendResponse(req.RequestID, 200, map[string]string{"status": "ok"})
		case "cancel":
			alias := req.Params["target"]
			s.app.CancelRelogin(ctx, alias)
			sndr.SendResponse(req.RequestID, 200, map[string]string{"status": "ok"})
		default:
			sndr.SendResponse(req.RequestID, 400, map[string]string{"error": "unknown command"})
		}
	}
}
