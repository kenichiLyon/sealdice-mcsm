package infra

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// MCSMClient handles communication with the MCSM API.
type MCSMClient struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// NewMCSMClient creates a new MCSM client.
func NewMCSMClient(baseURL, apiKey string) *MCSMClient {
	return &MCSMClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GenericResponse is the standard response wrapper.
type GenericResponse struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
	Time   int64       `json:"time"`
}

// DaemonInfo represents a remote node (daemon).
type DaemonInfo struct {
	UUID      string `json:"uuid"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	Available bool   `json:"available"`
	Remarks   string `json:"remarks"`
}

// InstanceDetail represents the instance information.
type InstanceDetail struct {
	InstanceUUID string `json:"instanceUuid"`
	Status       int    `json:"status"` // 0=Stopped, 1=Running, 2=Stopping, 3=Starting
	Config       struct {
		Nickname string `json:"nickname"`
		Type     string `json:"type"`
	} `json:"config"`
}

// InstanceListResponse is the response for listing instances.
type InstanceListResponse struct {
	Status int `json:"status"`
	Data   struct {
		Data []InstanceDetail `json:"data"`
	} `json:"data"`
}

// RemoteServicesResponse is the response for listing daemons.
type RemoteServicesResponse struct {
	Status int          `json:"status"`
	Data   []DaemonInfo `json:"data"`
}

// doRequest performs an HTTP request with the necessary headers and API key.
func (c *MCSMClient) doRequest(method, endpoint string, params map[string]string) ([]byte, error) {
	u, err := url.Parse(c.BaseURL + endpoint)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("apikey", c.APIKey)
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Required headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Check generic status in JSON
	var genResp GenericResponse
	if err := json.Unmarshal(body, &genResp); err == nil {
		if genResp.Status != 200 {
			return nil, fmt.Errorf("API returned error status %d: %s", genResp.Status, string(body))
		}
	}

	return body, nil
}

// FindInstanceDaemonID finds the daemon ID for a given instance UUID.
// It iterates through all available daemons.
func (c *MCSMClient) FindInstanceDaemonID(instanceID string) (string, error) {
	// 1. Get all daemons
	body, err := c.doRequest("GET", "/api/service/remote_services_system", nil)
	if err != nil {
		return "", fmt.Errorf("failed to list daemons: %w", err)
	}

	var daemonsResp RemoteServicesResponse
	if err := json.Unmarshal(body, &daemonsResp); err != nil {
		return "", fmt.Errorf("failed to parse daemons response: %w", err)
	}

	// 2. Search each daemon
	for _, daemon := range daemonsResp.Data {
		// List instances for this daemon
		// We use a large page_size to try to get all
		params := map[string]string{
			"daemonId":  daemon.UUID,
			"page":      "1",
			"page_size": "100",
		}
		body, err := c.doRequest("GET", "/api/service/remote_service_instances", params)
		if err != nil {
			log.Printf("[MCSM] Failed to list instances for daemon %s: %v", daemon.UUID, err)
			continue
		}

		var listResp InstanceListResponse
		if err := json.Unmarshal(body, &listResp); err != nil {
			continue
		}

		for _, inst := range listResp.Data.Data {
			if inst.InstanceUUID == instanceID {
				return daemon.UUID, nil
			}
		}
	}

	return "", fmt.Errorf("instance %s not found in any daemon", instanceID)
}

// InstanceAction sends a control command to an MCSM instance.
// Supported actions: start, stop, restart, kill.
func (c *MCSMClient) InstanceAction(instanceID, action string) error {
	daemonID, err := c.FindInstanceDaemonID(instanceID)
	if err != nil {
		return err
	}

	var endpoint string
	switch action {
	case "start":
		endpoint = "/api/protected_instance/open"
	case "stop":
		endpoint = "/api/protected_instance/stop"
	case "restart":
		endpoint = "/api/protected_instance/restart"
	case "kill":
		endpoint = "/api/protected_instance/kill"
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	params := map[string]string{
		"uuid":     instanceID,
		"daemonId": daemonID,
	}

	_, err = c.doRequest("GET", endpoint, params)
	return err
}

// StartInstance starts the instance.
func (c *MCSMClient) StartInstance(id string) error {
	return c.InstanceAction(id, "start")
}

// StopInstance stops the instance.
func (c *MCSMClient) StopInstance(id string) error {
	return c.InstanceAction(id, "stop")
}

// RestartInstance restarts the instance.
func (c *MCSMClient) RestartInstance(id string) error {
	return c.InstanceAction(id, "restart")
}

// ForceStopInstance force stops (kills) the instance.
func (c *MCSMClient) ForceStopInstance(id string) error {
	return c.InstanceAction(id, "kill")
}

// GetInstanceStatus returns the status of an instance.
func (c *MCSMClient) GetInstanceStatus(id string) (interface{}, error) {
	daemonID, err := c.FindInstanceDaemonID(id)
	if err != nil {
		return nil, err
	}

	// Use /api/instance to get details
	params := map[string]string{
		"uuid":     id,
		"daemonId": daemonID,
	}
	body, err := c.doRequest("GET", "/api/instance", params)
	if err != nil {
		return nil, err
	}

	var resp GenericResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetDashboard returns the system overview.
func (c *MCSMClient) GetDashboard() (interface{}, error) {
	// Trying /api/dashboard first
	body, err := c.doRequest("GET", "/api/dashboard", nil)
	if err == nil {
		var resp GenericResponse
		if err := json.Unmarshal(body, &resp); err == nil {
			return resp.Data, nil
		}
	}
	
	// Fallback to remote_services_system if dashboard fails
	return c.doRequest("GET", "/api/service/remote_services_system", nil)
}
