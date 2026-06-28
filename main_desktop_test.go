package main

import (
	"path/filepath"
	"testing"
)

func TestDesktopCommandServeUsesConfigFlag(t *testing.T) {
	cmd := parseDesktopCommand([]string{"serve", "-config", "/tmp/mc/config.json"})
	if cmd.mode != desktopModeServe {
		t.Fatalf("mode = %q, want %q", cmd.mode, desktopModeServe)
	}
	if cmd.configPath != "/tmp/mc/config.json" {
		t.Fatalf("configPath = %q, want /tmp/mc/config.json", cmd.configPath)
	}
}

func TestDesktopCommandDefaultOpensWindow(t *testing.T) {
	cmd := parseDesktopCommand(nil)
	if cmd.mode != desktopModeWindow {
		t.Fatalf("mode = %q, want %q", cmd.mode, desktopModeWindow)
	}
}

func TestDesktopConfigUsesAbsoluteDBPathBesideConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	dbPath := desktopDBPath(cfgPath)

	if !filepath.IsAbs(dbPath) {
		t.Fatalf("dbPath = %q, want absolute path", dbPath)
	}
	if filepath.Dir(dbPath) != filepath.Dir(cfgPath) {
		t.Fatalf("dbPath dir = %q, want %q", filepath.Dir(dbPath), filepath.Dir(cfgPath))
	}
}
