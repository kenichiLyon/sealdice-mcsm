package ws

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// MessageSender defines the interface for sending messages.
type MessageSender interface {
	Send(response Response) error
}

// Sender implements MessageSender for a WebSocket connection.
type Sender struct {
	Conn *websocket.Conn
	Mu   sync.Mutex
}

func (s *Sender) Send(resp Response) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return s.Conn.WriteJSON(resp)
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
