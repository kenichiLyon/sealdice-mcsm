package app

import (
	"context"
	"sync"

	"sealdice-mcsm/server/infra"
	"sealdice-mcsm/server/repo"
	"sealdice-mcsm/server/ws"
)

// Application implements the application logic.
type Application struct {
	Repo repo.Repository
	MCSM *infra.MCSMClient
	FS   *infra.QRCodeReader

	// FSM State
	fsmMu       sync.Mutex
	fsmState    FSMState
	fsmCancel   context.CancelFunc
	fsmSender   ws.MessageSender
	fsmAuthChan chan struct{} // Channel to receive .continue signal
}

// NewApplication creates a new application service.
func NewApplication(r repo.Repository, m *infra.MCSMClient, f *infra.QRCodeReader) *Application {
	return &Application{
		Repo:     r,
		MCSM:     m,
		FS:       f,
		fsmState: StateIdle,
	}
}

// Helper to resolve targets
func (app *Application) resolveTarget(target string) (string, error) {
	// Check if it's an alias
	binding, err := app.Repo.GetBinding(target)
	if err != nil {
		return "", err
	}
	if binding != nil {
		return binding.InstanceID, nil
	}
	// Assume it's an instance ID
	return target, nil
}
