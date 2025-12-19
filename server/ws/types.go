package ws

type Request struct {
	RequestID string            `json:"request_id"`
	Command   string            `json:"command"`
	Params    map[string]string `json:"params"`
}

type Response struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
	Code      int    `json:"code"`
	Message   string `json:"message,omitempty"`
	Data      any    `json:"data,omitempty"`
}

type Event struct {
	Type  string `json:"type"`
	Event string `json:"event"`
	Data  any    `json:"data,omitempty"`
}
