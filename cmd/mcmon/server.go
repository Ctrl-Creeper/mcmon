package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/lewiswu/mc-latency-monitor/internal/store"
)

//go:embed static
var staticFS embed.FS

var rangeToSeconds = map[string]int64{
	"1h": 3600, "6h": 6 * 3600, "12h": 12 * 3600,
	"1d": 24 * 3600, "7d": 7 * 24 * 3600, "30d": 30 * 24 * 3600,
}

func newMux(st *store.Store, cs *ConfigStore, mgr *Manager) *http.ServeMux {
	mux := http.NewServeMux()

	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	mux.HandleFunc("/api/targets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, cs.Targets())
		case http.MethodPost:
			var t Target
			if !decodeJSON(w, r, &t) {
				return
			}
			saved, err := cs.Upsert(t)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mgr.Start(saved)
			writeJSON(w, saved)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/targets/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/targets/")

		// Bulk import sub-route
		if id == "import" {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var targets []Target
			if !decodeJSON(w, r, &targets) {
				return
			}
			var saved []Target
			for _, t := range targets {
				s, err := cs.Upsert(t)
				if err != nil {
					http.Error(w, "target "+t.Name+": "+err.Error(), http.StatusBadRequest)
					return
				}
				mgr.Start(s)
				saved = append(saved, s)
			}
			writeJSON(w, saved)
			return
		}

		if id == "" {
			http.Error(w, "missing target id", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodPut:
			var t Target
			if !decodeJSON(w, r, &t) {
				return
			}
			t.ID = id
			saved, err := cs.Upsert(t)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mgr.Start(saved)
			writeJSON(w, saved)
		case http.MethodDelete:
			mgr.Stop(id)
			ok, err := cs.Delete(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/series", func(w http.ResponseWriter, r *http.Request) {
		targetID := r.URL.Query().Get("target")
		rangeKey := r.URL.Query().Get("range")
		secs, ok := rangeToSeconds[rangeKey]
		if !ok {
			secs = 3600
		}
		since := time.Now().Unix() - secs
		series, err := st.Series(targetID, since)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, series)
	})

	return mux
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
