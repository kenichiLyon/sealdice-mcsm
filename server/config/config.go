package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
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

type fileCfg struct {
	Addr                 string
	DBPath               string
	MCSMBaseURL          string
	MCSMApiKey           string
	ProtocolAlias        string
	SealdiceAlias        string
	QRFilePath           string
	PollIntervalSeconds  int
	QRWaitTimeoutSeconds int
	AuthTimeoutSeconds   int
	FSBase               string
}

func Load() *Config {
	cfg := &Config{}
	var cfgPath string
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
	flag.StringVar(&cfgPath, "config", "config.json", "json config file path")
	flag.Parse()
	b, err := os.ReadFile(cfgPath)
	if err == nil {
		var fc fileCfg
		if json.Unmarshal(b, &fc) == nil {
			if fc.Addr != "" {
				cfg.Addr = fc.Addr
			}
			if fc.DBPath != "" {
				cfg.DBPath = fc.DBPath
			}
			if fc.MCSMBaseURL != "" {
				cfg.MCSMBaseURL = fc.MCSMBaseURL
			}
			if fc.MCSMApiKey != "" {
				cfg.MCSMApiKey = fc.MCSMApiKey
			}
			if fc.ProtocolAlias != "" {
				cfg.ProtocolAlias = fc.ProtocolAlias
			}
			if fc.SealdiceAlias != "" {
				cfg.SealdiceAlias = fc.SealdiceAlias
			}
			if fc.QRFilePath != "" {
				cfg.QRFilePath = fc.QRFilePath
			}
			if fc.PollIntervalSeconds > 0 {
				cfg.PollInterval = time.Duration(fc.PollIntervalSeconds) * time.Second
			}
			if fc.QRWaitTimeoutSeconds > 0 {
				cfg.QRWaitTimeout = time.Duration(fc.QRWaitTimeoutSeconds) * time.Second
			}
			if fc.AuthTimeoutSeconds > 0 {
				cfg.AuthTimeout = time.Duration(fc.AuthTimeoutSeconds) * time.Second
			}
			cfg.FSBase = fc.FSBase
		}
	}
	if err != nil {
		fc := fileCfg{
			Addr:                 cfg.Addr,
			DBPath:               cfg.DBPath,
			MCSMBaseURL:          cfg.MCSMBaseURL,
			MCSMApiKey:           cfg.MCSMApiKey,
			ProtocolAlias:        cfg.ProtocolAlias,
			SealdiceAlias:        cfg.SealdiceAlias,
			QRFilePath:           cfg.QRFilePath,
			PollIntervalSeconds:  int(cfg.PollInterval.Seconds()),
			QRWaitTimeoutSeconds: int(cfg.QRWaitTimeout.Seconds()),
			AuthTimeoutSeconds:   int(cfg.AuthTimeout.Seconds()),
			FSBase:               cfg.FSBase,
		}
		data, _ := json.MarshalIndent(fc, "", "  ")
		_ = os.WriteFile(cfgPath, data, 0644)
		fmt.Println(string(data))
	}
	return cfg
}
