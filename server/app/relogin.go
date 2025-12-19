package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"sealdice-mcsm/server/ws"
)

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

// Relogin initiates the relogin process.
func (app *Application) Relogin(sender ws.MessageSender, params map[string]interface{}) error {
	target, ok := params["target"].(string)
	if !ok || target == "" {
		return fmt.Errorf("missing target")
	}

	// Resolve Protocol Instance ID
	protocolID, err := app.resolveTarget(target)
	if err != nil {
		return err
	}

	// Resolve Sealdice Instance ID (Optional or Convention)
	var sealdiceID string
	if st, ok := params["sealdice_target"].(string); ok && st != "" {
		sealdiceID, _ = app.resolveTarget(st)
	} else {
		// Try convention: target + "_sealdice"
		sid, err := app.resolveTarget(target + "_sealdice")
		if err == nil && sid != "" {
			sealdiceID = sid
		}
	}

	app.fsmMu.Lock()
	defer app.fsmMu.Unlock()

	if app.fsmState != StateIdle {
		return fmt.Errorf("relogin already in progress (state: %s)", app.fsmState)
	}

	// Initialize FSM
	ctx, cancel := context.WithCancel(context.Background())
	app.fsmCancel = cancel
	app.fsmState = StateRestartingProtocol
	app.fsmSender = sender
	app.fsmAuthChan = make(chan struct{})

	// Start FSM Goroutine
	go app.runReloginFSM(ctx, protocolID, sealdiceID)

	return nil
}

// ContinueRelogin signals the FSM to proceed after auth.
func (app *Application) ContinueRelogin() error {
	app.fsmMu.Lock()
	defer app.fsmMu.Unlock()

	if app.fsmState != StateWaitingForAuth {
		return fmt.Errorf("not waiting for auth (state: %s)", app.fsmState)
	}

	close(app.fsmAuthChan)
	return nil
}

// CancelRelogin cancels the current relogin process.
func (app *Application) CancelRelogin() error {
	app.fsmMu.Lock()
	defer app.fsmMu.Unlock()

	if app.fsmState == StateIdle {
		return nil
	}

	if app.fsmCancel != nil {
		app.fsmCancel()
	}
	app.resetFSM()
	return nil
}

func (app *Application) resetFSM() {
	app.fsmState = StateIdle
	app.fsmCancel = nil
	app.fsmSender = nil
	app.fsmAuthChan = nil
}

func (app *Application) runReloginFSM(ctx context.Context, protocolID, sealdiceID string) {
	defer func() {
		app.fsmMu.Lock()
		app.resetFSM()
		app.fsmMu.Unlock()
	}()

	sendError := func(msg string) {
		app.fsmMu.Lock()
		sender := app.fsmSender
		app.fsmMu.Unlock()
		if sender != nil {
			sender.Send(ws.Response{
				Code:    500,
				Message: msg,
			})
		}
	}
	
	sendInfo := func(msg string) {
		app.fsmMu.Lock()
		sender := app.fsmSender
		app.fsmMu.Unlock()
		if sender != nil {
			sender.Send(ws.Response{
				Code:    0,
				Message: msg,
			})
		}
	}

	log.Printf("[FSM] Starting Relogin for Protocol: %s, Sealdice: %s", protocolID, sealdiceID)

	// Step 1: Restart Protocol Instance
	if err := app.MCSM.RestartInstance(protocolID); err != nil {
		sendError(fmt.Sprintf("failed to restart protocol: %v", err))
		return
	}

	// Transition: WaitingForQRCode
	app.fsmMu.Lock()
	app.fsmState = StateWaitingForQRCode
	app.fsmMu.Unlock()
	sendInfo("Protocol restarting, waiting for QR code...")

	// Step 2: Poll for QR Code
	qrPath := "qrcode.png" 
	
	// Polling loop
	qrFound := false
	var qrContent string
	timeout := time.After(60 * time.Second) // Wait up to 60s for QR
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for !qrFound {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			sendError("timeout waiting for QR code")
			return
		case <-ticker.C:
			// Try to read
			if app.FS.FileExists(qrPath) {
				content, err := app.FS.ReadQRCode(qrPath)
				if err == nil && content != "" {
					qrContent = content
					qrFound = true
				}
			}
		}
	}

	// Transition: SendingQRCode
	app.fsmMu.Lock()
	app.fsmState = StateSendingQRCode
	app.fsmMu.Unlock()

	// Send QR Code
	app.fsmMu.Lock()
	sender := app.fsmSender
	app.fsmMu.Unlock()
	if sender != nil {
		sender.Send(ws.Response{
			Code:    0,
			Message: "QR Code received",
			Data: QRCodeEvent{
				ImageBase64: qrContent,
			},
		})
	}

	// Transition: WaitingForAuth
	app.fsmMu.Lock()
	app.fsmState = StateWaitingForAuth
	authChan := app.fsmAuthChan
	app.fsmMu.Unlock()
	sendInfo("Waiting for confirmation (.continue)...")

	// Step 3: Wait for .continue
	select {
	case <-ctx.Done():
		return
	case <-time.After(200 * time.Second):
		sendError("timeout waiting for confirmation")
		return
	case <-authChan:
		// User confirmed
	}

	// Transition: RestartingSealdice
	app.fsmMu.Lock()
	app.fsmState = StateRestartingSealdice
	app.fsmMu.Unlock()
	sendInfo("Confirmed. Restarting Sealdice...")

	// Step 4: Restart Sealdice
	if sealdiceID != "" {
		if err := app.MCSM.RestartInstance(sealdiceID); err != nil {
			sendError(fmt.Sprintf("failed to restart sealdice: %v", err))
			return
		}
	} else {
		log.Println("[FSM] No Sealdice instance ID provided, skipping restart.")
	}

	sendInfo("Relogin completed successfully.")
}
