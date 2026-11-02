package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Mode string

const (
	ModeNormal   Mode = "normal"
	ModeBalanced Mode = "balanced"
	ModeMax      Mode = "max"
)

func ParseMode(s string) (Mode, error) {
	switch Mode(s) {
	case ModeNormal, ModeBalanced, ModeMax:
		return Mode(s), nil
	default:
		return "", fmt.Errorf("invalid mode: %s (normal|balanced|max)", s)
	}
}

type AgentConfig struct {
	ListenAddr   string   `json:"listen_addr"`   
	Token        string   `json:"token"`         
	XMRigPath    string   `json:"xmrig_path"`    
	XMRigVersion string   `json:"xmrig_version"` 
	WorkDir      string   `json:"work_dir"`      
	PoolURLs     []string `json:"pool_urls"`      
	Wallet       string   `json:"wallet"`         
	WorkerName   string   `json:"worker_name"`    
	TLS          bool     `json:"tls"`            
	DonateLevel  int      `json:"donate_level"`   
	Mode         Mode     `json:"mode"`           
	ForceGPU     string   `json:"force_gpu"`      
	APIPort      int      `json:"xmrig_api_port"`
}

func LoadOrDefault(path string) (*AgentConfig, error) {
	cfg := &AgentConfig{
		ListenAddr:  "0.0.0.0:9090",
		XMRigPath:   "./xmrig",
		WorkDir:     "./miner-run",
		DonateLevel: 1,
		Mode:        ModeNormal,
		APIPort:     16000,
	}
	if path == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}
	return cfg, nil
}

func (c *AgentConfig) Save(path string) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
