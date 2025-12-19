package app

import (
	"context"
	"fmt"
	"sync"

	"sealdice-mcsm/internal/infra/fs"
	"sealdice-mcsm/internal/infra/mcsm"
	"sealdice-mcsm/internal/model"
	"sealdice-mcsm/internal/repo"
)

// MessageSender defines the interface for sending messages back to the client.
type MessageSender interface {
	Send(response model.Response) error
}

// Service implements the application logic.
type Service struct {
	Repo repo.Repository
	MCSM *mcsm.Client
	FS   *fs.FileReader

	// FSM State
	fsmMu       sync.Mutex
	fsmState    model.FSMState
	fsmCancel   context.CancelFunc
	fsmSender   MessageSender
	fsmAuthChan chan struct{} // Channel to receive .continue signal
}

// NewService creates a new application service.
func NewService(r repo.Repository, m *mcsm.Client, f *fs.FileReader) *Service {
	return &Service{
		Repo:     r,
		MCSM:     m,
		FS:       f,
		fsmState: model.StateIdle,
	}
}

// Bind binds an instance ID to an alias.
func (s *Service) Bind(alias, instanceID string) error {
	return s.Repo.SaveBinding(alias, instanceID)
}

// resolveTarget resolves an alias or instance ID to an instance ID.
func (s *Service) resolveTarget(target string) (string, error) {
	// Check if it's an alias
	binding, err := s.Repo.GetBinding(target)
	if err != nil {
		return "", err
	}
	if binding != nil {
		return binding.InstanceID, nil
	}
	// Assume it's an instance ID
	return target, nil
}

// Control performs an action on an instance.
func (s *Service) Control(target, action string) error {
	id, err := s.resolveTarget(target)
	if err != nil {
		return err
	}

	switch action {
	case "start":
		return s.MCSM.StartInstance(id)
	case "stop":
		return s.MCSM.StopInstance(id)
	case "restart":
		return s.MCSM.RestartInstance(id)
	case "fstop":
		return s.MCSM.ForceStopInstance(id)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// Status returns the status of an instance or dashboard.
func (s *Service) Status(target string) (interface{}, error) {
	if target == "" {
		// Return MCSM Dashboard
		return s.MCSM.GetDashboard()
	}

	id, err := s.resolveTarget(target)
	if err != nil {
		return nil, err
	}
	statusData, err := s.MCSM.GetInstanceStatus(id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"target": target,
		"id":     id,
		"data":   statusData,
	}, nil
}
