package config

import (
	"flag"
	"time"
)

type Config struct {
	Addr          string
	DBPath        string
	MCSMBaseURL   string
	MCSMApiKey    string
	ProtocolAlias string
	SealdiceAlias string
	QRFilePath    string
	PollInterval  time.Duration
	QRWaitTimeout time.Duration
	AuthTimeout   time.Duration
	FSBase        string
}

func Load() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.Addr, "addr", ":8088", "websocket server address")
	flag.StringVar(&cfg.DBPath, "db", "server.db", "sqlite db path for alias bindings")
	flag.StringVar(&cfg.MCSMBaseURL, "mcsm", "http://127.0.0.1:23333", "MCSM panel base URL")
	flag.StringVar(&cfg.MCSMApiKey, "apikey", "", "MCSM API key")
	flag.StringVar(&cfg.ProtocolAlias, "protocol-alias", "dice-protocol", "alias for protocol instance")
	flag.StringVar(&cfg.SealdiceAlias, "sealdice-alias", "dice-main", "alias for sealdice instance")
	flag.StringVar(&cfg.QRFilePath, "qrfile", "qrcode.png", "fixed QR filename under protocol instance workspace")
	flag.DurationVar(&cfg.PollInterval, "poll", 2*time.Second, "poll interval for QR detection")
	flag.DurationVar(&cfg.QRWaitTimeout, "qr-timeout", 200*time.Second, "timeout waiting for QR generation")
	flag.DurationVar(&cfg.AuthTimeout, "auth-timeout", 200*time.Second, "timeout waiting for user auth (.continue)")
	flag.StringVar(&cfg.FSBase, "fs-base", "", "local filesystem base for instance workspaces (optional)")
	flag.Parse()
	return cfg
}
