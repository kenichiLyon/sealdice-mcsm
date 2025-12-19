package infra

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

func (c *Client) Dashboard() (map[string]any, error) {
	b, err := c.do(http.MethodGet, "/api/dashboard", nil)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) InstanceAction(instanceID, daemonID, action string) error {
	p := fmt.Sprintf("/api/protected_instance/command/%s/%s", daemonID, instanceID)
	_, err := c.do(http.MethodPost, p, map[string]string{"command": action})
	return err
}

func (c *Client) InstanceDetail(instanceID, daemonID string) (map[string]any, error) {
	p := fmt.Sprintf("/api/protected_instance/detail/%s/%s", daemonID, instanceID)
	b, err := c.do(http.MethodGet, p, nil)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}
