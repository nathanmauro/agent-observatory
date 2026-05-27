package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Addr     string         `json:"addr"`
	DBPath   string         `json:"db_path"`
	Agents   map[string]AgentConfig `json:"agents"`
}

type AgentConfig struct {
	Enabled bool   `json:"enabled"`
	Root    string `json:"root,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		Addr:   "127.0.0.1:3284",
		DBPath: "observatory.db",
		Agents: map[string]AgentConfig{
			"claude":  {Enabled: true},
			"codex":   {Enabled: true},
			"augment": {Enabled: true},
			"cursor":  {Enabled: true},
		},
	}
}

func Load(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		path = defaultPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func defaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "agent-observatory", "config.json")
}
