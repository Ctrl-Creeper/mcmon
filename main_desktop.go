package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ctrl-Creeper/mcmon/internal/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

type DesktopApp struct {
	ctx context.Context
}

type desktopMode string

const (
	desktopModeWindow desktopMode = "window"
	desktopModeServe  desktopMode = "serve"
)

type desktopCommand struct {
	mode       desktopMode
	configPath string
}

func (a *DesktopApp) startup(ctx context.Context) {
	a.ctx = ctx
}

func main() {
	cmd := parseDesktopCommand(os.Args[1:])
	if cmd.mode == desktopModeServe {
		ensureDesktopConfig(cmd.configPath)
		if err := app.RunServer(context.Background(), cmd.configPath); err != nil {
			log.Fatal(err)
		}
		return
	}

	configPath := cmd.configPath
	ensureDesktopConfig(configPath)
	rt, err := app.NewRuntime(configPath)
	if err != nil {
		log.Fatal(err)
	}
	defer rt.Close()

	desktop := &DesktopApp{}
	err = wails.Run(&options.App{
		Title:             "mcmon",
		Width:             1180,
		Height:            820,
		MinWidth:          900,
		MinHeight:         620,
		HideWindowOnClose: true,
		BackgroundColour:  options.NewRGB(247, 247, 249),
		AssetServer: &assetserver.Options{
			Assets:  app.StaticFS(),
			Handler: rt.Handler,
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, "/api/") {
						rt.Handler.ServeHTTP(w, r)
						return
					}
					next.ServeHTTP(w, r)
				})
			},
		},
		OnStartup: desktop.startup,
		Bind:      []interface{}{desktop},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func parseDesktopCommand(args []string) desktopCommand {
	defaultConfigPath, err := desktopConfigPath()
	if err != nil {
		log.Fatal(err)
	}
	if len(args) > 0 && args[0] == "serve" {
		fs := flag.NewFlagSet("mcmon serve", flag.ExitOnError)
		configPath := fs.String("config", defaultConfigPath, "path to config file")
		fs.Parse(args[1:])
		return desktopCommand{mode: desktopModeServe, configPath: *configPath}
	}
	return desktopCommand{mode: desktopModeWindow, configPath: defaultConfigPath}
}

func desktopConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "mcmon")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func ensureDesktopConfig(configPath string) {
	b, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		writeDesktopConfig(configPath, desktopDBPath(configPath))
		return
	}
	if err != nil {
		log.Fatal(err)
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		log.Fatal(err)
	}
	if dbPath, _ := raw["db_path"].(string); filepath.IsAbs(dbPath) {
		return
	}
	raw["db_path"] = desktopDBPath(configPath)
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(configPath, out, 0644); err != nil {
		log.Fatal(err)
	}
}

func writeDesktopConfig(configPath, dbPath string) {
	raw := map[string]any{
		"listen_addr": "127.0.0.1:8090",
		"db_path":     dbPath,
		"targets": []map[string]any{
			{
				"id":               "example",
				"name":             "Example Server",
				"host":             "mc.example.com",
				"port":             25565,
				"interval_sec":     60,
				"timeout_ms":       1500,
				"probes_per_burst": 5,
				"probe_gap_ms":     1500,
				"protocol_version": 760,
				"monitors": map[string]any{
					"online":  map[string]any{"enabled": true, "interval_sec": 60},
					"players": map[string]any{"enabled": true, "interval_sec": 60},
					"latency": map[string]any{"enabled": true, "interval_sec": 60, "probes_per_burst": 5, "probe_gap_ms": 1500, "protocol_version": 760},
					"loss":    map[string]any{"enabled": true, "interval_sec": 60, "probes_per_burst": 5, "probe_gap_ms": 1500},
				},
			},
		},
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(configPath, out, 0644); err != nil {
		log.Fatal(err)
	}
}

func desktopDBPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "mcmon.db")
}
