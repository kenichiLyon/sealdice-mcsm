package app

import (
	"context"
	"errors"
	"time"

	"sealdice-mcsm/server/config"
	"sealdice-mcsm/server/infra"
)

type ReloginFSM struct {
	cfg    *config.Config
	alias  string
	client *infra.Client
	fs     infra.FileAPI
	sender Sender

	restartAt time.Time
	state     ReloginStatus
	continueC chan struct{}
	cancelC   chan struct{}
}

func NewReloginFSM(cfg *config.Config, alias string, client *infra.Client, fs infra.FileAPI, sender Sender) *ReloginFSM {
	return &ReloginFSM{
		cfg:       cfg,
		alias:     alias,
		client:    client,
		fs:        fs,
		sender:    sender,
		state:     StatusIdle,
		continueC: make(chan struct{}, 1),
		cancelC:   make(chan struct{}, 1),
	}
}

func (f *ReloginFSM) Run(ctx context.Context) error {
	// Step 1: restart protocol instance
	f.state = StatusRestartingProtocol
	_ = f.client.InstanceAction(f.alias, "local", "restart")
	f.restartAt = time.Now()

	// Step 2-5: poll QR file
	f.state = StatusWaitingQRCode
	timeout := time.NewTimer(f.cfg.QRWaitTimeout)
	ticker := time.NewTicker(f.cfg.PollInterval)
	defer ticker.Stop()
	defer timeout.Stop()
	for {
		select {
		case <-ctx.Done():
			f.state = StatusFailed
			return ctx.Err()
		case <-f.cancelC:
			f.state = StatusFailed
			return errors.New("relogin canceled")
		case <-ticker.C:
			mt, exists, err := f.fs.FileStat(f.alias, f.cfg.QRFilePath)
			if err != nil {
				// keep polling; an error shouldn't abort immediately
				continue
			}
			if exists && mt.After(f.restartAt) {
				f.state = StatusSendingQRCode
				b64, err := f.fs.FileReadBase64(f.alias, f.cfg.QRFilePath)
				if err != nil {
					continue
				}
				f.sender.SendEvent("qrcode_ready", map[string]any{
					"alias":        f.alias,
					"generated_at": time.Now().UTC().Format(time.RFC3339),
					"qrcode":       b64,
				})
				f.state = StatusWaitingAuth
				goto WAIT_AUTH
			}
		case <-timeout.C:
			f.state = StatusFailed
			return errors.New("qrcode timeout")
		}
	}

WAIT_AUTH:
	authTimeout := time.NewTimer(f.cfg.AuthTimeout)
	defer authTimeout.Stop()
	for {
		select {
		case <-ctx.Done():
			f.state = StatusFailed
			return ctx.Err()
		case <-f.cancelC:
			f.state = StatusFailed
			return errors.New("relogin canceled")
		case <-f.continueC:
			// Step 8: restart sealdice instance
			f.state = StatusRestartingSealdice
			_ = f.client.InstanceAction(f.cfg.SealdiceAlias, "local", "restart")
			f.state = StatusFinished
			return nil
		case <-authTimeout.C:
			f.state = StatusFailed
			return errors.New("auth wait timeout")
		}
	}
}

func (f *ReloginFSM) Continue() {
	select {
	case f.continueC <- struct{}{}:
	default:
	}
}

func (f *ReloginFSM) Cancel() {
	select {
	case f.cancelC <- struct{}{}:
	default:
	}
}
