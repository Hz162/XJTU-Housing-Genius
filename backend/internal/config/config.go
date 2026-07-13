package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type ConfigFile struct {
	// Housing-specific config will be added as needed
}

var (
	cfg     ConfigFile
	cfgPath string
	mu      sync.RWMutex
)

func init() {
	exe, _ := os.Executable()
	cfgPath = filepath.Join(filepath.Dir(exe), "housing-config.json")
	Load()
}

func Path() string { return cfgPath }

func Load() {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		cfg = ConfigFile{}
		return
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		cfg = ConfigFile{}
	}
}

func Save() {
	mu.RLock()
	data, err := json.MarshalIndent(cfg, "", "  ")
	mu.RUnlock()
	if err != nil {
		return
	}
	os.WriteFile(cfgPath, data, 0644)
}

func Get() ConfigFile {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}
