// mcmon is a self-contained Minecraft server latency monitor: it pings
// configured servers using the Java Edition status protocol, stores
// per-minute latency samples in SQLite, and serves a small web UI with
// graphs (smokeping-style min/median/max bands and packet loss). Targets
// and their polling interval are fully editable at runtime via the API/UI,
// and "install"/"uninstall" subcommands register it as a background
// service so it keeps running across reboots/logins.
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/lewiswu/mc-latency-monitor/internal/app"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			configPath := commandConfigPath("mcmon install", os.Args[2:])
			if err := app.InstallBackground(configPath); err != nil {
				log.Fatalf("install: %v", err)
			}
			return
		case "uninstall":
			if err := app.UninstallBackground(); err != nil {
				log.Fatalf("uninstall: %v", err)
			}
			return
		case "serve":
			configPath := commandConfigPath("mcmon serve", os.Args[2:])
			if err := app.RunServer(context.Background(), configPath); err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	fs := flag.NewFlagSet("mcmon", flag.ExitOnError)
	configPath := fs.String("config", "config.json", "path to config file (created with defaults if missing)")
	fs.Parse(os.Args[1:])

	if err := app.RunServer(context.Background(), *configPath); err != nil {
		log.Fatal(err)
	}
}

func commandConfigPath(name string, args []string) string {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	configPath := fs.String("config", "config.json", "path to config file")
	fs.Parse(args)
	return *configPath
}
