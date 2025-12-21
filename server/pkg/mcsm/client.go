package mcsm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

type Client struct {
	Base   string
	APIKey string
	HTTP   *http.Client
}

func NewClient(base, apikey string) *Client {
	return &Client{
		Base:   base,
		APIKey: apikey,
		HTTP:   &http.Client{Timeout: 15 * time.Second},
	}
}

// Response Types
type DashboardResponse struct {
	Status int `json:"status"`
	Data   struct {
		Version     string `json:"version"`
		RemoteCount struct {
			Total int `json:"total"`
		} `json:"remoteCount"`
	} `json:"data"`
}

type InstanceDetailResponse struct {
	Status int `json:"status"`
	Data   struct {
		InstanceUUID string `json:"instanceUuid"`
		Status       int    `json:"status"`
		Process      struct {
			CpuUsage float64 `json:"cpuUsage"`
			Memory   int64   `json:"memory"`
		} `json:"process"`
		Config struct {
			Nickname string `json:"nickname"`
		} `json:"config"`
	} `json:"data"`
}

func (c *Client) do(method, p string, body any) ([]byte, error) {
	u, err := url.Parse(c.Base)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, p)
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, u.String(), rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("apikey", c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
	}
	return b, nil
}

func (c *Client) Dashboard() (*DashboardResponse, error) {
	b, err := c.do(http.MethodGet, "/api/dashboard", nil)
	if err != nil {
		return nil, err
	}
	var out DashboardResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) InstanceAction(instanceID, daemonID, action string) error {
	// action: start, stop, restart, kill
	p := fmt.Sprintf("/api/protected_instance/command/%s/%s", daemonID, instanceID)
	_, err := c.do(http.MethodGet, p+"?command="+action, nil) 
	// Note: Documentation says POST for some, GET for others?
	// Web Reference 4 says:
	// 启动实例: GET /api/protected_instance/open
	// 停止实例: GET /api/protected_instance/stop
	// 重启实例: GET /api/protected_instance/restart
	// 强制结束: GET /api/protected_instance/kill
	// 发送命令: GET /api/protected_instance/command
	
	// The original code used POST to /api/protected_instance/command with {"command": action}
	// But web ref says GET for open/stop/restart.
	// I will follow the web reference for specific actions if possible, OR check if "command" endpoint supports "start/stop".
	// The original code used: `/api/protected_instance/command` with `command` body. 
	// Web ref says `GET /api/protected_instance/command` takes `command` param. 
	// Let's assume standard endpoints:
	
	var endpoint string
	switch action {
	case "start":
		endpoint = "open"
	case "stop":
		endpoint = "stop"
	case "restart":
		endpoint = "restart"
	case "kill", "fstop":
		endpoint = "kill"
	default:
		// Fallback to sending command
		p := fmt.Sprintf("/api/protected_instance/command?uuid=%s&daemonId=%s&command=%s", instanceID, daemonID, action)
		_, err := c.do(http.MethodGet, p, nil)
		return err
	}
	
	p = fmt.Sprintf("/api/protected_instance/%s?uuid=%s&daemonId=%s", endpoint, instanceID, daemonID)
	_, err = c.do(http.MethodGet, p, nil)
	return err
}

func (c *Client) InstanceDetail(instanceID, daemonID string) (*InstanceDetailResponse, error) {
	p := fmt.Sprintf("/api/instance?uuid=%s&daemonId=%s", instanceID, daemonID)
	b, err := c.do(http.MethodGet, p, nil)
	if err != nil {
		return nil, err
	}
	var out InstanceDetailResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
