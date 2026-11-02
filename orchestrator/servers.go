package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Server struct {
	Name  string `json:"name"` 
	Addr  string `json:"addr"` 
	Token string `json:"token"`
	OS    string `json:"os"` 
}

type ServerList struct {
	Servers []Server `json:"servers"`
}

func LoadServers(path string) (*ServerList, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ServerList{Servers: []Server{}}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var sl ServerList
	if err := json.Unmarshal(b, &sl); err != nil {
		return nil, fmt.Errorf("invalid servers.json format: %w", err)
	}
	return &sl, nil
}

func saveServers(path string, sl *ServerList) error {
	b, err := json.MarshalIndent(sl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

func (s Server) call(method, path string, body interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s%s", s.Addr, path)
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if s.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.Token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach %s: %w", s.Name, err)
	}
	defer resp.Body.Close()

	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("invalid response from %s: %w", s.Name, err)
	}
	if resp.StatusCode >= 300 {
		return out, fmt.Errorf("%s returned: %v", s.Name, out["error"])
	}
	return out, nil
}

func (s Server) Status() (map[string]interface{}, error) {
	return s.call(http.MethodGet, "/status", nil)
}

func (s Server) SetMode(mode string) (map[string]interface{}, error) {
	return s.call(http.MethodPost, "/mode", map[string]string{"mode": mode})
}

func (s Server) Start() (map[string]interface{}, error) {
	return s.call(http.MethodPost, "/start", nil)
}

func (s Server) Stop() (map[string]interface{}, error) {
	return s.call(http.MethodPost, "/stop", nil)
}

func (s Server) Logs(lines int) (map[string]interface{}, error) {
	return s.call(http.MethodGet, fmt.Sprintf("/logs?lines=%d", lines), nil)
}
