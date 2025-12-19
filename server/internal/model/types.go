package model

// Binding represents the mapping between an alias and an MCSM instance ID.
type Binding struct {
	Alias      string `json:"alias"`
	InstanceID string `json:"instance_id"`
}

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

// FSMState represents the state of the Relogin FSM.
type FSMState int

const (
	StateIdle              FSMState = iota
	StateRestartingProtocol         // Restarting the protocol adapter (e.g., Lagrange)
	StateWaitingForQRCode           // Waiting for the QR code image to appear
	StateSendingQRCode              // Sending QR code to client (transient state usually)
	StateWaitingForAuth             // Waiting for user confirmation (.continue)
	StateRestartingSealdice         // Restarting the main Sealdice instance
)

func (s FSMState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateRestartingProtocol:
		return "RestartingProtocol"
	case StateWaitingForQRCode:
		return "WaitingForQRCode"
	case StateSendingQRCode:
		return "SendingQRCode"
	case StateWaitingForAuth:
		return "WaitingForAuth"
	case StateRestartingSealdice:
		return "RestartingSealdice"
	default:
		return "Unknown"
	}
}

// QRCodeEvent is the data payload when sending a QR code to the client.
type QRCodeEvent struct {
	ImageBase64 string `json:"image_base64"`
}
