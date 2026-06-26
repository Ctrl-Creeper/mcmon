package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

type Target struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	IntervalSec     int    `json:"interval_sec"`      // how often to run a probe burst
	TimeoutMs       int    `json:"timeout_ms"`        // per-probe connect/read timeout
	ProbesPerBurst  int    `json:"probes_per_burst"`  // samples taken per burst
	ProbeGapMs      int    `json:"probe_gap_ms"`      // delay between samples within a burst
	ProtocolVersion int    `json:"protocol_version"`
}

func (t Target) normalized() Target {
	if t.IntervalSec <= 0 {
		t.IntervalSec = 60
	}
	if t.TimeoutMs <= 0 {
		t.TimeoutMs = 1500
	}
	if t.ProbesPerBurst <= 0 {
		t.ProbesPerBurst = 5
	}
	if t.ProbeGapMs < 0 {
		t.ProbeGapMs = 0
	}
	if t.ProtocolVersion == 0 {
		t.ProtocolVersion = 760
	}
	return t
}

func (t Target) validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(t.Host) == "" {
		return fmt.Errorf("host is required")
	}
	if t.Port <= 0 || t.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

type Config struct {
	ListenAddr string   `json:"listen_addr"`
	DBPath     string   `json:"db_path"`
	Targets    []Target `json:"targets"`
}

func defaultConfig() Config {
	return Config{
		ListenAddr: ":8090",
		DBPath:     "mcmon.db",
		Targets: []Target{
			{
				ID: "example", Name: "Example Server",
				Host: "mc.example.com", Port: 25565,
				IntervalSec: 60, TimeoutMs: 1500, ProbesPerBurst: 5, ProbeGapMs: 1500,
				ProtocolVersion: 760,
			},
		},
	}
}

// ConfigStore is the mutable, persisted source of truth for targets. All
// reads/writes go through it so the REST API and the prober manager see a
// consistent view and edits survive a restart.
type ConfigStore struct {
	mu   sync.Mutex
	path string
	cfg  Config
}

func openConfigStore(path string) (*ConfigStore, error) {
	cfg, err := loadConfig(path)
	if err != nil {
		return nil, err
	}
	for i := range cfg.Targets {
		cfg.Targets[i] = cfg.Targets[i].normalized()
	}
	return &ConfigStore{path: path, cfg: cfg}, nil
}

func (c *ConfigStore) Snapshot() Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.cfg
	out.Targets = append([]Target(nil), c.cfg.Targets...)
	return out
}

func (c *ConfigStore) Targets() []Target {
	return c.Snapshot().Targets
}

func (c *ConfigStore) Get(id string) (Target, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, t := range c.cfg.Targets {
		if t.ID == id {
			return t, true
		}
	}
	return Target{}, false
}

// Upsert validates and normalizes t, then inserts or replaces it by ID,
// persisting the result to disk. Returns the stored value.
func (c *ConfigStore) Upsert(t Target) (Target, error) {
	if err := t.validate(); err != nil {
		return Target{}, err
	}
	t = t.normalized()

	c.mu.Lock()
	defer c.mu.Unlock()
	replaced := false
	for i, existing := range c.cfg.Targets {
		if existing.ID == t.ID {
			c.cfg.Targets[i] = t
			replaced = true
			break
		}
	}
	if !replaced {
		c.cfg.Targets = append(c.cfg.Targets, t)
	}
	if err := writeConfig(c.path, c.cfg); err != nil {
		return Target{}, err
	}
	return t, nil
}

func (c *ConfigStore) Delete(id string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, t := range c.cfg.Targets {
		if t.ID == id {
			c.cfg.Targets = append(c.cfg.Targets[:i], c.cfg.Targets[i+1:]...)
			if err := writeConfig(c.path, c.cfg); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func loadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := defaultConfig()
		if werr := writeConfig(path, cfg); werr != nil {
			return cfg, werr
		}
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func writeConfig(path string, cfg Config) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
