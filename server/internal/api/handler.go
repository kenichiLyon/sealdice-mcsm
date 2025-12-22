package api

import (
	"fmt"
	"log"
	"net/http"

	"sealdice-mcsm/server/config"
	"sealdice-mcsm/server/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Handler struct {
	Svc *service.Service
	Cfg *config.Config
}

func NewHandler(svc *service.Service, cfg *config.Config) *Handler {
	return &Handler{Svc: svc, Cfg: cfg}
}

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.Cfg.Auth.Enable {
			c.Next()
			return
		}
		token := c.GetHeader("Authorization")
		if token != h.Cfg.Auth.Token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func (h *Handler) SetupRoutes(r *gin.Engine) {
	r.GET("/public/*filepath", func(c *gin.Context) {
		c.File("./temp/" + c.Param("filepath"))
	})

	wsGroup := r.Group("/ws")
	wsGroup.Use(h.AuthMiddleware())
	wsGroup.GET("", h.HandleWS)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSNotifier adapts *websocket.Conn to Notifier interface
type WSNotifier struct {
	Conn  *websocket.Conn
	ReqID string
}

func (w *WSNotifier) SendEvent(event string, data any) error {
	return w.Conn.WriteJSON(gin.H{
		"type":   "event",
		"event":  event,
		"data":   data,
		"req_id": w.ReqID, // Inject ReqID
	})
}

func (h *Handler) HandleWS(c *gin.Context) {
	// Fallback auth check for query param
	if h.Cfg.Auth.Enable {
		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.Query("token")
		}
		if token != h.Cfg.Auth.Token {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WS Upgrade Error:", err)
		return
	}
	defer conn.Close()

	// Handle connection
	for {
		var req struct {
			Action  string            `json:"action"`
			Command string            `json:"command"`
			ReqID   string            `json:"req_id"`
			Params  map[string]string `json:"params"`
		}
		if err := conn.ReadJSON(&req); err != nil {
			break
		}

		notifier := &WSNotifier{Conn: conn, ReqID: req.ReqID}

		action := req.Action
		if action == "" {
			action = req.Command
		}

		var res any
		var errOp error

		switch action {
		case "bind":
			// Params: alias, protocol_id, core_id
			errOp = h.Svc.InstanceSvc.Bind(req.Params["alias"], req.Params["protocol_id"], req.Params["core_id"])
			res = map[string]string{"status": "ok"}
		case "unbind":
			errOp = h.Svc.InstanceSvc.Unbind(req.Params["alias"])
			res = map[string]string{"status": "ok"}
		case "get_binding":
			res, errOp = h.Svc.InstanceSvc.GetByAlias(req.Params["alias"])

		case "start", "stop", "restart", "fstop", "kill":
			// Need to resolve alias to instance ID?
			// Old logic had "role" param.
			// New logic: If alias provided, which instance?
			// Requirement says: "Control instances".
			// If alias is provided, we probably need to know which one (protocol or core).
			// Or maybe the params include "instance_id" directly?
			// Let's support both.

			target := req.Params["target"]
			if target == "" {
				target = req.Params["alias"]
			}

			// Simple logic: if target looks like UUID, use it.
			// If it's an alias, we need 'role' (protocol/core).
			role := req.Params["role"]
			instanceID := target

			if len(target) <= 20 { // Likely alias
				binding, err := h.Svc.InstanceSvc.GetByAlias(target)
				if err != nil {
					errOp = err
				} else {
					if role == "protocol" {
						instanceID = binding.ProtocolInstanceID
					} else if role == "core" {
						instanceID = binding.CoreInstanceID
					} else {
						// Default? or Error?
						errOp = fmt.Errorf("role required for alias target")
					}
				}
			}

			if errOp == nil {
				errOp = h.Svc.MCSM.InstanceAction(instanceID, "local", action)
				res = map[string]string{"status": "ok"}
			}

		case "status":
			// Similar resolution logic
			target := req.Params["target"]
			if target == "" {
				res, errOp = h.Svc.MCSM.Dashboard()
			} else {
				instanceID := target
				if len(target) <= 20 {
					binding, err := h.Svc.InstanceSvc.GetByAlias(target)
					if err != nil {
						errOp = err
					} else {
						role := req.Params["role"]
						if role == "protocol" {
							instanceID = binding.ProtocolInstanceID
						} else {
							instanceID = binding.CoreInstanceID
						}
					}
				}
				if errOp == nil {
					res, errOp = h.Svc.MCSM.InstanceDetail(instanceID, "local")
				}
			}

		case "relogin":
			// Async workflow
			alias := req.Params["alias"]
			if alias == "" {
				alias = req.Params["target"]
			}
			go func() {
				if err := h.Svc.WorkflowSvc.Relogin(alias, notifier); err != nil {
					notifier.SendEvent("error", map[string]string{
						"alias": alias,
						"msg":   err.Error(),
					})
				}
			}()
			res = map[string]string{"status": "started"}

		case "continue":
			alias := req.Params["alias"]
			if alias == "" {
				alias = req.Params["target"]
			}
			errOp = h.Svc.WorkflowSvc.Continue(alias)
			res = map[string]string{"status": "signal_sent"}

		default:
			errOp = fmt.Errorf("unknown command: %s", action)
		}

		resp := gin.H{
			"req_id": req.ReqID,
			"type":   "response",
			"data":   res,
		}
		if errOp != nil {
			resp["type"] = "error"
			resp["message"] = errOp.Error()
			resp["code"] = 500
		} else {
			resp["code"] = 200
		}

		conn.WriteJSON(resp)
	}
}
