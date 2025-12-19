package ws

import (
	"log"
	"sync"

	"sealdice-mcsm/internal/app"
	"sealdice-mcsm/internal/model"

	"github.com/gorilla/websocket"
)

// Handler handles a single WebSocket connection.
type Handler struct {
	Conn   *websocket.Conn
	App    *app.Service
	SendMu sync.Mutex
}

// NewHandler creates a new WebSocket handler.
func NewHandler(conn *websocket.Conn, app *app.Service) *Handler {
	return &Handler{
		Conn: conn,
		App:  app,
	}
}

// Send implements app.MessageSender.
func (h *Handler) Send(resp model.Response) error {
	h.SendMu.Lock()
	defer h.SendMu.Unlock()
	return h.Conn.WriteJSON(resp)
}

// Loop starts the read loop for the connection.
func (h *Handler) Loop() {
	defer func() {
		h.App.CancelRelogin() // Cancel any ongoing FSM if connection drops
		h.Conn.Close()
	}()

	for {
		var req model.Request
		if err := h.Conn.ReadJSON(&req); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Read error: %v", err)
			}
			break
		}

		h.handleRequest(req)
	}
}

func (h *Handler) handleRequest(req model.Request) {
	var resp model.Response
	resp.RequestID = req.RequestID

	var err error
	var data interface{}

	switch req.Command {
	case "bind":
		alias, _ := req.Params["alias"].(string)
		instanceID, _ := req.Params["instance_id"].(string)
		if alias == "" || instanceID == "" {
			// Try "target" as instanceID? No, bind needs both.
			err = logAndError("missing alias or instance_id")
		} else {
			err = h.App.Bind(alias, instanceID)
			data = "bound"
		}

	case "start", "stop", "restart", "fstop":
		target, _ := req.Params["target"].(string)
		if target == "" {
			err = logAndError("missing target")
		} else {
			err = h.App.Control(target, req.Command)
			data = "ok"
		}

	case "status":
		target, _ := req.Params["target"].(string)
		// target can be empty for dashboard
		data, err = h.App.Status(target)

	case "relogin":
		// This triggers async FSM. We return "accepted" immediately.
		err = h.App.Relogin(h, req.Params)
		if err == nil {
			data = "relogin started"
		}

	case "continue":
		// User confirms waiting relogin
		err = h.App.ContinueRelogin()
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
	h.Send(resp)
}

func logAndError(msg string) error {
	log.Printf("[WS] Error: %s", msg)
	return &AppError{Msg: msg}
}

type AppError struct {
	Msg string
}

func (e *AppError) Error() string {
	return e.Msg
}
