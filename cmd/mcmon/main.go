// mcmon is a self-contained Minecraft server latency monitor: it pings
// configured servers using the Java Edition status protocol, stores
// per-minute latency samples in SQLite, and serves a small web UI with
// graphs (smokeping-style min/median/max bands and packet loss).
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/lewiswu/mc-latency-monitor/internal/store"
)

func main() {
	configPath := flag.String("config", "config.json", "path to config file (created with defaults if missing)")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	startProbeLoop(st, cfg.Targets)

	mux := newMux(st, cfg.Targets)
	log.Printf("mcmon listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatal(err)
	}
}
