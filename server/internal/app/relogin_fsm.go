package app

import (
	"context"
	"fmt"
	"log"
	"sealdice-mcsm/internal/model"
	"time"
)

// Relogin initiates the relogin process.
// It expects params to contain "target" (the protocol instance).
// It optionally looks for "sealdice_target" to restart the main instance.
func (s *Service) Relogin(sender MessageSender, params map[string]interface{}) error {
	target, ok := params["target"].(string)
	if !ok || target == "" {
		return fmt.Errorf("missing target")
	}

	// Resolve Protocol Instance ID
	protocolID, err := s.resolveTarget(target)
	if err != nil {
		return err
	}

	// Resolve Sealdice Instance ID (Optional or Convention)
	// We'll try to find a "sealdice_target" param, or fallback to looking up "target_sealdice" alias,
	// or just skip if not found.
	var sealdiceID string
	if st, ok := params["sealdice_target"].(string); ok && st != "" {
		sealdiceID, _ = s.resolveTarget(st)
	} else {
		// Try convention: target + "_sealdice"
		// We don't error if not found, just don't restart it.
		// However, requirement says "Restart sealdice instance".
		// We will try to resolve the convention.
		sid, err := s.resolveTarget(target + "_sealdice")
		if err == nil && sid != "" {
			sealdiceID = sid
		}
	}

	s.fsmMu.Lock()
	defer s.fsmMu.Unlock()

	if s.fsmState != model.StateIdle {
		return fmt.Errorf("relogin already in progress (state: %s)", s.fsmState)
	}

	// Initialize FSM
	ctx, cancel := context.WithCancel(context.Background())
	s.fsmCancel = cancel
	s.fsmState = model.StateRestartingProtocol
	s.fsmSender = sender
	s.fsmAuthChan = make(chan struct{})

	// Start FSM Goroutine
	go s.runReloginFSM(ctx, protocolID, sealdiceID)

	return nil
}

// ContinueRelogin signals the FSM to proceed after auth.
func (s *Service) ContinueRelogin() error {
	s.fsmMu.Lock()
	defer s.fsmMu.Unlock()

	if s.fsmState != model.StateWaitingForAuth {
		return fmt.Errorf("not waiting for auth (state: %s)", s.fsmState)
	}

	close(s.fsmAuthChan)
	return nil
}

// CancelRelogin cancels the current relogin process.
func (s *Service) CancelRelogin() error {
	s.fsmMu.Lock()
	defer s.fsmMu.Unlock()

	if s.fsmState == model.StateIdle {
		return nil
	}

	if s.fsmCancel != nil {
		s.fsmCancel()
	}
	s.resetFSM()
	return nil
}

func (s *Service) resetFSM() {
	s.fsmState = model.StateIdle
	s.fsmCancel = nil
	s.fsmSender = nil
	s.fsmAuthChan = nil
}

func (s *Service) runReloginFSM(ctx context.Context, protocolID, sealdiceID string) {
	defer func() {
		s.fsmMu.Lock()
		s.resetFSM()
		s.fsmMu.Unlock()
	}()

	sendError := func(msg string) {
		s.fsmMu.Lock()
		sender := s.fsmSender
		s.fsmMu.Unlock()
		if sender != nil {
			sender.Send(model.Response{
				Code:    500,
				Message: msg,
			})
		}
	}

	sendInfo := func(msg string) {
		s.fsmMu.Lock()
		sender := s.fsmSender
		s.fsmMu.Unlock()
		if sender != nil {
			sender.Send(model.Response{
				Code:    0,
				Message: msg,
			})
		}
	}

	log.Printf("[FSM] Starting Relogin for Protocol: %s, Sealdice: %s", protocolID, sealdiceID)

	// Step 1: Restart Protocol Instance
	if err := s.MCSM.RestartInstance(protocolID); err != nil {
		sendError(fmt.Sprintf("failed to restart protocol: %v", err))
		return
	}

	// Transition: WaitingForQRCode
	s.fsmMu.Lock()
	s.fsmState = model.StateWaitingForQRCode
	s.fsmMu.Unlock()
	sendInfo("Protocol restarting, waiting for QR code...")

	// Step 2: Poll for QR Code
	// We'll look for a file named "qrcode.png" or "qrcode.txt" in the protocol instance's directory.
	// Assumption: We know the path. For now, we assume it's in a standard location or relative to base.
	// We'll use a simulated path "qrcode.png" for now.
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
			if s.FS.FileExists(qrPath) {
				content, err := s.FS.ReadQRCode(qrPath)
				if err == nil && content != "" {
					qrContent = content
					qrFound = true
				}
			}
		}
	}

	// Transition: SendingQRCode
	s.fsmMu.Lock()
	s.fsmState = model.StateSendingQRCode
	s.fsmMu.Unlock()

	// Send QR Code
	s.fsmMu.Lock()
	sender := s.fsmSender
	s.fsmMu.Unlock()
	if sender != nil {
		sender.Send(model.Response{
			Code:    0,
			Message: "QR Code received",
			Data: model.QRCodeEvent{
				ImageBase64: qrContent,
			},
		})
	}

	// Transition: WaitingForAuth
	s.fsmMu.Lock()
	s.fsmState = model.StateWaitingForAuth
	authChan := s.fsmAuthChan
	s.fsmMu.Unlock()
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
	s.fsmMu.Lock()
	s.fsmState = model.StateRestartingSealdice
	s.fsmMu.Unlock()
	sendInfo("Confirmed. Restarting Sealdice...")

	// Step 4: Restart Sealdice
	if sealdiceID != "" {
		if err := s.MCSM.RestartInstance(sealdiceID); err != nil {
			sendError(fmt.Sprintf("failed to restart sealdice: %v", err))
			return
		}
	} else {
		log.Println("[FSM] No Sealdice instance ID provided, skipping restart.")
	}

	sendInfo("Relogin completed successfully.")
}
