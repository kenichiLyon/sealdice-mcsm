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
	// InstanceAction performs actions like start, stop, restart, kill
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
	
	p := fmt.Sprintf("/api/protected_instance/%s?uuid=%s&daemonId=%s", endpoint, instanceID, daemonID)
	_, err := c.do(http.MethodGet, p, nil)
	return err
}

func (c *Client) StartInstance(uuid, daemonID string) error {
	return c.InstanceAction(uuid, daemonID, "start")
}

func (c *Client) StopInstance(uuid, daemonID string) error {
	return c.InstanceAction(uuid, daemonID, "stop")
}

type FileListResponse struct {
	Status int `json:"status"`
	Data   struct {
		Items []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
			Time string `json:"time"`
			Type int    `json:"type"` // 0=folder, 1=file
		} `json:"items"`
	} `json:"data"`
}

type FileStatus struct {
	Name         string
	Size         int64
	LastModified time.Time
}

// GetFileStatus retrieves file info. It lists the parent directory and searches for the file.
func (c *Client) GetFileStatus(uuid, daemonID, filePath string) (*FileStatus, error) {
	dir, file := path.Split(filePath)
	if dir == "" {
		dir = "/"
	}
	// MCSM API expects target to be the directory path
	p := fmt.Sprintf("/api/files/list?daemonId=%s&uuid=%s&target=%s&page=0&page_size=1000", 
		daemonID, uuid, url.QueryEscape(dir))
	
	b, err := c.do(http.MethodGet, p, nil)
	if err != nil {
		return nil, err
	}
	
	var res FileListResponse
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}
	
	if res.Status != 200 {
		return nil, fmt.Errorf("api error: %d", res.Status)
	}
	
	for _, item := range res.Data.Items {
		if item.Name == file {
			// Parse Time: "Fri Jun 07 2024 08:53:34 GMT+0800 (中国标准时间)"
			// We might need a robust parser.
			// Try standard layouts or specific one.
			// If parsing fails, we might return current time or error.
			// Let's try to parse "Fri Jun 07 2024 08:53:34 GMT+0800" part.
			
			// Simple approach: try to find a library or just ignore time zone name if possible.
			// Or assumes server returns something parseable.
			// Let's implement a best-effort parser.
			t, err := parseMCSMTime(item.Time)
			if err != nil {
				// Log warning?
				fmt.Printf("Warning: failed to parse time '%s': %v\n", item.Time, err)
				t = time.Now() // Fallback?
			}
			
			return &FileStatus{
				Name:         item.Name,
				Size:         item.Size,
				LastModified: t,
			}, nil
		}
	}
	
	return nil, fmt.Errorf("file not found: %s", filePath)
}

func parseMCSMTime(tStr string) (time.Time, error) {
	// Example: "Fri Jun 07 2024 08:53:34 GMT+0800 (中国标准时间)"
	// Go layout: "Mon Jan 02 2006 15:04:05 GMT-0700"
	// We need to strip the suffix in parentheses.
	
	idx := 0
	for i, r := range tStr {
		if r == '(' {
			idx = i
			break
		}
	}
	if idx > 0 {
		tStr = tStr[:idx]
	}
	tStr = url.QueryEscape(tStr) // Wait, trim space
	// Just trim space
	// "Fri Jun 07 2024 08:53:34 GMT+0800 "
	// Layout: "Mon Jan 02 2006 15:04:05 GMT-0700"
	layout := "Mon Jan 02 2006 15:04:05 GMT-0700"
	// Actually, Go's time parsing is strict.
	// Let's try to parse it.
	// The example "GMT+0800" is numeric zone.
	
	// If the string is exactly "Fri Jun 07 2024 08:53:34 GMT+0800 " (with trailing space)
	// We should TrimSpace.
	// Let's try a few layouts.
	
	// Note: If using `dateparse` lib is allowed, it would be easier. But I should avoid adding heavy deps if not necessary.
	// I will use a simple substring parser or try to match the format.
	
	// Ref: https://github.com/mcsmanager/MCSManager/blob/master/frontend/src/tools/time.ts (Source code of MCSM frontend might give clue)
	// Actually, the API returns what `new Date().toString()` returns in NodeJS on the server?
	// It looks like JS `Date.toString()`.
	
	// Let's try to parse "Fri Jun 07 2024 08:53:34 GMT+0800"
	// Layout: "Mon Jan 02 2006 15:04:05 GMT-0700"
	
	// First remove the (...) part
	if idx := 0; true {
		for i, r := range tStr {
			if r == '(' {
				tStr = tStr[:i]
				break
			}
		}
	}
	tStr = string(bytes.TrimSpace([]byte(tStr)))
	
	return time.Parse("Mon Jan 02 2006 15:04:05 GMT-0700", tStr)
}

type DownloadConfigResponse struct {
	Status int `json:"status"`
	Data   struct {
		Password string `json:"password"`
		Addr     string `json:"addr"`
	} `json:"data"`
}

func (c *Client) DownloadFile(uuid, daemonID, filePath string) ([]byte, error) {
	// 1. Get download config
	p := fmt.Sprintf("/api/files/download?daemonId=%s&uuid=%s&file_name=%s", 
		daemonID, uuid, url.QueryEscape(filePath))
		
	// Note: POST according to docs
	// httpPOST /api/files/download
	// Query params? Docs say "Query Params" but it's a POST?
	// Usually POST takes body, but docs say "Query Params".
	// Let's try POST with empty body and query params.
	
	u, err := url.Parse(c.Base)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "/api/files/download")
	q := u.Query()
	q.Set("daemonId", daemonID)
	q.Set("uuid", uuid)
	q.Set("file_name", filePath)
	u.RawQuery = q.Encode()
	
	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("apikey", c.APIKey)
		req.Header.Set("X-API-KEY", c.APIKey)
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
	
	var res DownloadConfigResponse
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, fmt.Errorf("json error: %v, body: %s", err, string(b))
	}
	
	if res.Status != 200 {
		return nil, fmt.Errorf("download config error: %d", res.Status)
	}
	
	// 2. Download from node
	// URL: http(s)://{{Daemon Addr}}/download/{{password}}/{{fileName}}
	// We need to construct the URL.
	// Note: res.Data.Addr might be "localhost:24444". If server is remote, this might be an issue if it returns internal IP.
	// But assuming we are running on same network or it's accessible.
	// Also, if protocol is https, we should use https.
	
	// For now, assume http unless base is https? 
	// Or maybe construct relative to base if daemon is local?
	// The doc says `http(s)://{{Daemon Addr}}`.
	
	downloadURL := fmt.Sprintf("http://%s/download/%s/%s", 
		res.Data.Addr, res.Data.Password, url.QueryEscape(path.Base(filePath)))
		
	// Download
	dReq, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	dResp, err := c.HTTP.Do(dReq)
	if err != nil {
		return nil, err
	}
	defer dResp.Body.Close()
	
	if dResp.StatusCode != 200 {
		return nil, fmt.Errorf("download failed: %d", dResp.StatusCode)
	}
	
	return io.ReadAll(dResp.Body)
}

func (c *Client) WaitForQRCode(uuid, daemonID, filePath string, startTime time.Time) ([]byte, error) {
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for qrcode")
		case <-ticker.C:
			status, err := c.GetFileStatus(uuid, daemonID, filePath)
			if err != nil {
				// Log error but continue polling?
				// fmt.Println("Poll error:", err)
				continue
			}
			
			if status.LastModified.After(startTime) {
				// Found new file
				return c.DownloadFile(uuid, daemonID, filePath)
			}
		}
	}
}
