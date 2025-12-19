package app

import "time"

type ControlAction string

const (
	ActionStart   ControlAction = "start"
	ActionStop    ControlAction = "stop"
	ActionRestart ControlAction = "restart"
	ActionFStop   ControlAction = "fstop"
)

type ReloginStatus string

const (
	StatusIdle              ReloginStatus = "idle"
	StatusRestartingProtocol ReloginStatus = "restarting_protocol"
	StatusWaitingQRCode     ReloginStatus = "waiting_qrcode"
	StatusSendingQRCode     ReloginStatus = "sending_qrcode"
	StatusWaitingAuth       ReloginStatus = "waiting_auth"
	StatusRestartingSealdice ReloginStatus = "restarting_sealdice"
	StatusFinished          ReloginStatus = "finished"
	StatusFailed            ReloginStatus = "failed"
)

type QRReadyEvent struct {
	Alias       string    `json:"alias"`
	GeneratedAt time.Time `json:"generated_at"`
	QRBase64    string    `json:"qrcode"`
}

