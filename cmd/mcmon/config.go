package main

import (
	"encoding/json"
	"os"
)

type Target struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	TimeoutMs       int    `json:"timeout_ms"`
	ProbesPerMinute int    `json:"probes_per_minute"`
	ProbeIntervalMs int    `json:"probe_interval_ms"`
	ProtocolVersion int    `json:"protocol_version"`
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
				TimeoutMs: 1500, ProbesPerMinute: 5, ProbeIntervalMs: 1500,
				ProtocolVersion: 760,
			},
		},
	}
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
