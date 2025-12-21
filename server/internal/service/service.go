package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sealdice-mcsm/server/config"
	"sealdice-mcsm/server/internal/data"
	"sealdice-mcsm/server/pkg/mcsm"
)

type Service struct {
	Cfg  *config.Config
	Repo data.BindingRepo
	MCSM *mcsm.Client
}

func NewService(cfg *config.Config, repo data.BindingRepo, mcsm *mcsm.Client) *Service {
	// Ensure temp directory exists
	_ = os.MkdirAll("./temp", 0755)
	return &Service{
		Cfg:  cfg,
		Repo: repo,
		MCSM: mcsm,
	}
}

func (s *Service) SaveTempFile(data []byte, ext string) (string, error) {
	// Generate filename
	hash := md5.Sum(data)
	filename := hex.EncodeToString(hash[:]) + ext
	path := filepath.Join("./temp", filename)

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}

	// Schedule deletion
	go func() {
		time.Sleep(5 * time.Minute)
		os.Remove(path)
	}()

	// Construct URL
	// Assuming /public/ mapped to ./temp/
	baseURL := s.Cfg.App.ExternalURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost%s", s.Cfg.Server.Port)
	}
	// Trim trailing slash
	if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	
	return fmt.Sprintf("%s/public/%s", baseURL, filename), nil
}

// Logic for Binding, Control, etc. can be added here
func (s *Service) Bind(alias, role, instanceID string) error {
	return s.Repo.SaveBinding(alias, role, instanceID)
}

func (s *Service) Control(target, role, action string) error {
	var instanceID string
	var err error
	
	if len(target) > 10 {
		instanceID = target
	} else {
		instanceID, err = s.Repo.GetBinding(target, role)
		if err != nil {
			return err
		}
	}
	
	return s.MCSM.InstanceAction(instanceID, "local", action) // Assuming daemonID "local"
}

func (s *Service) Status(target, role string) (any, error) {
	if target == "" {
		return s.MCSM.Dashboard()
	}
	
	var instanceID string
	var err error
	if len(target) > 10 {
		instanceID = target
	} else {
		instanceID, err = s.Repo.GetBinding(target, role)
		if err != nil {
			return nil, err
		}
	}
	return s.MCSM.InstanceDetail(instanceID, "local")
}
