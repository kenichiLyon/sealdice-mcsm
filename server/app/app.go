package app

import (
	"context"
	"errors"
	"sync"

	"sealdice-mcsm/server/config"
	"sealdice-mcsm/server/infra"
	"sealdice-mcsm/server/repo"
)

type Sender interface {
	SendResponse(requestID string, code int, data any)
	SendEvent(name string, data any)
}

type Application struct {
	cfg  *config.Config
	repo repo.BindingRepo
	mcsm *infra.Client
	fs   infra.FileAPI

	mu   sync.Mutex
	fsms map[string]*ReloginFSM
}

func NewApplication(cfg *config.Config, repo repo.BindingRepo, mcsm *infra.Client, fs infra.FileAPI) *Application {
	return &Application{
		cfg:  cfg,
		repo: repo,
		mcsm: mcsm,
		fs:   fs,
		fsms: make(map[string]*ReloginFSM),
	}
}

func (a *Application) Bind(_ context.Context, alias, instanceID string) error {
	return a.repo.SaveBinding(alias, instanceID)
}

func (a *Application) resolve(target string) (string, error) {
	if target == "" {
		return "", errors.New("target required")
	}
	// accept instance id directly
	if len(target) > 10 {
		return target, nil
	}
	id, err := a.repo.GetBinding(target)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (a *Application) Control(ctx context.Context, target string, action ControlAction) error {
	instanceID, err := a.resolve(target)
	if err != nil {
		return err
	}
	// daemon id discovery is omitted; expect "local" or a fixed daemon id if needed
	daemonID := "local"
	return a.mcsm.InstanceAction(instanceID, daemonID, string(action))
}

func (a *Application) Status(ctx context.Context, target string) (any, error) {
	if target == "" {
		return a.mcsm.Dashboard()
	}
	instanceID, err := a.resolve(target)
	if err != nil {
		return nil, err
	}
	daemonID := "local"
	return a.mcsm.InstanceDetail(instanceID, daemonID)
}

func (a *Application) Relogin(ctx context.Context, sender Sender, requestID string, alias string) error {
	a.mu.Lock()
	if _, exists := a.fsms[alias]; exists {
		a.mu.Unlock()
		return errors.New("relogin already in progress for alias")
	}
	fsm := NewReloginFSM(a.cfg, alias, a.mcsm, a.fs, sender)
	a.fsms[alias] = fsm
	a.mu.Unlock()
	sender.SendResponse(requestID, 200, map[string]string{"status": string(StatusWaitingQRCode)})
	go func() {
		defer func() {
			a.mu.Lock()
			delete(a.fsms, alias)
			a.mu.Unlock()
		}()
		_ = fsm.Run(ctx)
	}()
	return nil
}

func (a *Application) ContinueRelogin(_ context.Context, alias string) {
	a.mu.Lock()
	fsm := a.fsms[alias]
	a.mu.Unlock()
	if fsm != nil {
		fsm.Continue()
	}
}

func (a *Application) CancelRelogin(_ context.Context, alias string) {
	a.mu.Lock()
	fsm := a.fsms[alias]
	a.mu.Unlock()
	if fsm != nil {
		fsm.Cancel()
	}
}
