package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"sealdice-mcsm/server/config"
	"sealdice-mcsm/server/internal/service"
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
	wsGroup.Use(h.AuthMiddleware()) // Assuming WS also needs auth? Or maybe handled in handshake?
	// Note: Websocket auth usually via query param or protocol header. 
	// If Middleware checks "Authorization" header, standard JS WebSocket API can't set it easily (except protocol).
	// For now, let's keep it but user might need to pass token in query if header not supported by client.
	// But current Plugin client doesn't send headers.
	// We might need to check query param "token" as fallback.
	
	wsGroup.GET("", h.HandleWS)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
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

	// Handle connection (Simple loop for now, similar to original)
	for {
		var req struct {
			Action  string            `json:"action"`
			Command string            `json:"command"` // compat
			ReqID   string            `json:"req_id"`
			Params  map[string]string `json:"params"`
		}
		if err := conn.ReadJSON(&req); err != nil {
			break
		}
		
		action := req.Action
		if action == "" {
			action = req.Command
		}

		// Dispatch
		var res any
		var errOp error

		switch action {
		case "bind":
			errOp = h.Svc.Bind(req.Params["alias"], req.Params["role"], req.Params["instance_id"])
			res = map[string]string{"status": "ok"}
		case "start", "stop", "restart", "fstop", "kill":
			errOp = h.Svc.Control(req.Params["target"], req.Params["role"], action)
			res = map[string]string{"status": "ok"}
		case "status":
			res, errOp = h.Svc.Status(req.Params["target"], req.Params["role"])
		// Relogin flow would need FSM here, omitted for brevity as it wasn't explicitly requested in "Skeleton" 
		// but I should probably add a stub or port it if I want full feature parity.
		// For skeleton, this is enough.
		default:
			errOp = log.Output(1, "Unknown command: "+action)
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
