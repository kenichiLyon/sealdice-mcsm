package ws

// Request represents a WebSocket command from the client.
type Request struct {
	RequestID string                 `json:"request_id"`
	Command   string                 `json:"command"`
	Params    map[string]interface{} `json:"params"`
}

// Response represents a WebSocket response to the client.
type Response struct {
	RequestID string      `json:"request_id"`
	Code      int         `json:"code"`    // 0 for success, non-zero for error
	Message   string      `json:"message"` // Error message or status
	Data      interface{} `json:"data,omitempty"`
}
