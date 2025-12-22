package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"sealdice-mcsm/server/pkg/mcsm"
)

type Notifier interface {
	SendEvent(event string, data any) error
}

type WorkflowService struct {
	InstanceSvc *InstanceService
	CommonSvc   *Service // For SaveTempFile
	MCSM        *mcsm.Client
	
	// Map alias -> channel for signaling "continue"
	pendingLogins sync.Map // map[string]chan struct{}
}

func NewWorkflowService(instSvc *InstanceService, commonSvc *Service, mcsm *mcsm.Client) *WorkflowService {
	return &WorkflowService{
		InstanceSvc: instSvc,
		CommonSvc:   commonSvc,
		MCSM:        mcsm,
	}
}

func (s *WorkflowService) Relogin(alias string, notifier Notifier) error {
	// 1. Check Binding
	binding, err := s.InstanceSvc.GetByAlias(alias)
	if err != nil {
		return fmt.Errorf("alias %s not bound", alias)
	}

	// Prevent concurrent relogins for same alias
	if _, loaded := s.pendingLogins.LoadOrStore(alias, make(chan struct{})); loaded {
		return fmt.Errorf("relogin already in progress for %s", alias)
	}
	
	// Ensure cleanup
	defer s.pendingLogins.Delete(alias)

	// 2. Restart Protocol Instance
	log.Printf("[%s] Restarting Protocol Instance: %s", alias, binding.ProtocolInstanceID)
	// TODO: DaemonID "local" assumption? 
	// If stored in binding, better. But for now assume "local" or fetch from binding if we added it.
	// Schema didn't have DaemonID. Assume "local" or fixed.
	daemonID := "local" 
	
	startTime := time.Now()
	if err := s.MCSM.StartInstance(binding.ProtocolInstanceID, daemonID); err != nil {
		// Try restart if start fails? Or just RestartInstance?
		// My client has RestartInstance fallback.
		// Let's use Restart.
		if err := s.MCSM.InstanceAction(binding.ProtocolInstanceID, daemonID, "restart"); err != nil {
			return fmt.Errorf("failed to restart protocol: %v", err)
		}
	}
	
	notifier.SendEvent("log", fmt.Sprintf("Protocol instance restarted. Waiting for QR code..."))

	// 3. Wait for QRCode
	// Assuming QR code file path is fixed or configurable?
	// Usually "qrcode.png" in the instance root.
	qrPath := "qrcode.png" 
	
	log.Printf("[%s] Waiting for QR Code...", alias)
	qrData, err := s.MCSM.WaitForQRCode(binding.ProtocolInstanceID, daemonID, qrPath, startTime)
	if err != nil {
		return fmt.Errorf("failed to get QR code: %v", err)
	}

	// 4. Save to Static Storage & Push
	url, err := s.CommonSvc.SaveTempFile(qrData, ".png")
	if err != nil {
		return fmt.Errorf("failed to save QR image: %v", err)
	}
	
	log.Printf("[%s] QR Code ready: %s", alias, url)
	notifier.SendEvent("qrcode", map[string]string{
		"alias": alias,
		"url":   url,
	})
	notifier.SendEvent("log", "Please scan the QR code to login.")

	// 5. Wait for "continue" signal
	log.Printf("[%s] Waiting for user confirmation...", alias)
	signalCh, _ := s.pendingLogins.Load(alias)
	ch := signalCh.(chan struct{})
	
	select {
	case <-ch:
		log.Printf("[%s] Received continue signal.", alias)
		notifier.SendEvent("log", "Login confirmed. Restarting Core...")
	case <-time.After(3 * time.Minute):
		return fmt.Errorf("timeout waiting for user confirmation")
	}

	// 6. Restart Core Instance
	if err := s.MCSM.InstanceAction(binding.CoreInstanceID, daemonID, "restart"); err != nil {
		return fmt.Errorf("failed to restart core: %v", err)
	}
	
	log.Printf("[%s] Relogin sequence completed.", alias)
	notifier.SendEvent("success", "Relogin completed successfully.")
	
	return nil
}

func (s *WorkflowService) Continue(alias string) error {
	val, ok := s.pendingLogins.Load(alias)
	if !ok {
		return fmt.Errorf("no active relogin process for %s", alias)
	}
	
	ch := val.(chan struct{})
	
	// Non-blocking send to avoid deadlock if receiver is gone (though Load check helps)
	select {
	case ch <- struct{}{}:
		return nil
	default:
		return fmt.Errorf("process already signaled or stuck")
	}
}
