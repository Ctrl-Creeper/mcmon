// mcmon is a self-contained Minecraft server latency monitor: it pings
// configured servers using the Java Edition status protocol, stores
// per-minute latency samples in SQLite, and serves a small web UI with
// graphs (smokeping-style min/median/max bands and packet loss). Targets
// and their polling interval are fully editable at runtime via the API/UI,
// and "install"/"uninstall" subcommands register it as a background
// service so it keeps running across reboots/logins.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/lewiswu/mc-latency-monitor/internal/store"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			if err := installService(); err != nil {
				log.Fatalf("install: %v", err)
			}
			return
		case "uninstall":
			if err := uninstallService(); err != nil {
				log.Fatalf("uninstall: %v", err)
			}
			return
		}
	}

	fs := flag.NewFlagSet("mcmon", flag.ExitOnError)
	configPath := fs.String("config", "config.json", "path to config file (created with defaults if missing)")
	fs.Parse(os.Args[1:])

	cs, err := openConfigStore(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	st, err := store.Open(cs.Snapshot().DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	mgr := NewManager(st)
	mgr.Sync(cs.Targets())

	mux := newMux(st, cs, mgr)
	addr := cs.Snapshot().ListenAddr
	log.Printf("mcmon listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
