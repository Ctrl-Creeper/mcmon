package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

const defaultProtocolVersion = 760

type Target struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Host            string   `json:"host"`
	Port            int      `json:"port"`
	TimeoutMs       int      `json:"timeout_ms"` // shared default timeout
	Monitors        Monitors `json:"monitors"`
	IntervalSec     int      `json:"interval_sec"`     // how often to run a probe burst
	ProbesPerBurst  int      `json:"probes_per_burst"` // samples taken per burst
	ProbeGapMs      int      `json:"probe_gap_ms"`     // delay between samples within a burst
	ProtocolVersion int      `json:"protocol_version"`
}

type Monitors struct {
	Online  SimpleMonitor `json:"online"`
	Players SimpleMonitor `json:"players"`
	Latency ProbeMonitor  `json:"latency"`
	Loss    ProbeMonitor  `json:"loss"`
}

type SimpleMonitor struct {
	Enabled     bool `json:"enabled"`
	IntervalSec int  `json:"interval_sec"`
}

type ProbeMonitor struct {
	Enabled         bool `json:"enabled"`
	IntervalSec     int  `json:"interval_sec"`
	ProbesPerBurst  int  `json:"probes_per_burst"`
	ProbeGapMs      int  `json:"probe_gap_ms"`
	ProtocolVersion int  `json:"protocol_version,omitempty"`
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
		t.ProtocolVersion = defaultProtocolVersion
	}
	t.Monitors = t.normalizedMonitors()
	return t
}

func (t Target) normalizedMonitors() Monitors {
	interval := t.IntervalSec
	if interval <= 0 {
		interval = 60
	}
	burst := t.ProbesPerBurst
	if burst <= 0 {
		burst = 5
	}
	gap := t.ProbeGapMs
	if gap < 0 {
		gap = 0
	}
	proto := t.ProtocolVersion
	if proto == 0 {
		proto = defaultProtocolVersion
	}

	m := t.Monitors
	if m.Online.IntervalSec <= 0 {
		m.Online.IntervalSec = interval
	}
	if m.Players.IntervalSec <= 0 {
		m.Players.IntervalSec = interval
	}
	if m.Latency.IntervalSec <= 0 {
		m.Latency.IntervalSec = interval
	}
	if m.Latency.ProbesPerBurst <= 0 {
		m.Latency.ProbesPerBurst = burst
	}
	if m.Latency.ProbeGapMs <= 0 {
		m.Latency.ProbeGapMs = gap
	}
	if m.Latency.ProtocolVersion == 0 {
		m.Latency.ProtocolVersion = proto
	}
	if m.Loss.IntervalSec <= 0 {
		m.Loss.IntervalSec = interval
	}
	if m.Loss.ProbesPerBurst <= 0 {
		m.Loss.ProbesPerBurst = burst
	}
	if m.Loss.ProbeGapMs <= 0 {
		m.Loss.ProbeGapMs = gap
	}
	if m.Loss.ProtocolVersion == 0 {
		m.Loss.ProtocolVersion = proto
	}

	if !t.hasExplicitMonitors() {
		m.Online.Enabled = true
		m.Players.Enabled = true
		m.Latency.Enabled = true
		m.Loss.Enabled = true
	}
	return m
}

func (t Target) hasExplicitMonitors() bool {
	return t.Monitors.Online.Enabled ||
		t.Monitors.Players.Enabled ||
		t.Monitors.Latency.Enabled ||
		t.Monitors.Loss.Enabled ||
		t.Monitors.Online.IntervalSec > 0 ||
		t.Monitors.Players.IntervalSec > 0 ||
		t.Monitors.Latency.IntervalSec > 0 ||
		t.Monitors.Loss.IntervalSec > 0
}

func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (t Target) validate() error {
	if strings.TrimSpace(t.Host) == "" {
		return fmt.Errorf("host is required")
	}
	if t.Port <= 0 || t.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

type Config struct {
	ListenAddr       string   `json:"listen_addr"`
	DBPath           string   `json:"db_path"`
	RemoteHost       string   `json:"remote_host,omitempty"`
	RemoteAdminToken string   `json:"remote_admin_token,omitempty"`
	Targets          []Target `json:"targets"`
}

func defaultConfig() Config {
	return Config{
		ListenAddr: "127.0.0.1:8090",
		DBPath:     "mcmon.db",
		Targets: []Target{
			{
				ID: "example", Name: "Example Server",
				Host: "mc.example.com", Port: 25565,
				IntervalSec: 60, TimeoutMs: 1500, ProbesPerBurst: 5, ProbeGapMs: 1500,
				ProtocolVersion: defaultProtocolVersion,
				Monitors: Monitors{
					Online:  SimpleMonitor{Enabled: true, IntervalSec: 60},
					Players: SimpleMonitor{Enabled: true, IntervalSec: 60},
					Latency: ProbeMonitor{Enabled: true, IntervalSec: 60, ProbesPerBurst: 5, ProbeGapMs: 1500, ProtocolVersion: defaultProtocolVersion},
					Loss:    ProbeMonitor{Enabled: true, IntervalSec: 60, ProbesPerBurst: 5, ProbeGapMs: 1500, ProtocolVersion: defaultProtocolVersion},
				},
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
	dirty := false
	for i := range cfg.Targets {
		cfg.Targets[i] = cfg.Targets[i].normalized()
		if strings.TrimSpace(cfg.Targets[i].ID) == "" {
			cfg.Targets[i].ID = generateID()
			dirty = true
		}
	}
	if dirty {
		_ = writeConfig(path, cfg)
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
	if strings.TrimSpace(t.ID) == "" {
		t.ID = generateID()
	}

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

func (c *ConfigStore) RemoteHost() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg.RemoteHost
}

func (c *ConfigStore) RemoteConfig() (hostURL, adminToken string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg.RemoteHost, c.cfg.RemoteAdminToken
}

func (c *ConfigStore) SetRemoteConfig(hostURL, adminToken string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfg.RemoteHost = hostURL
	c.cfg.RemoteAdminToken = adminToken
	return writeConfig(c.path, c.cfg)
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
