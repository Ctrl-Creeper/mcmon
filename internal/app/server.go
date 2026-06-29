package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/YOUR_PATH/mc-latency-monitor/internal/store"
)

//go:embed static
var staticFS embed.FS

var rangeToSeconds = map[string]int64{
	"1h": 3600, "6h": 6 * 3600, "12h": 12 * 3600,
	"1d": 24 * 3600, "7d": 7 * 24 * 3600, "30d": 30 * 24 * 3600,
}

func newMux(st *store.Store, cs *ConfigStore, mgr *Manager, configPath string) *http.ServeMux {
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

	mux.HandleFunc("/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		targetID := r.URL.Query().Get("target")
		metric := r.URL.Query().Get("metric")
		if strings.TrimSpace(targetID) == "" || strings.TrimSpace(metric) == "" {
			http.Error(w, "target and metric are required", http.StatusBadRequest)
			return
		}
		rangeKey := r.URL.Query().Get("range")
		secs, ok := rangeToSeconds[rangeKey]
		if !ok {
			secs = 3600
		}
		since := time.Now().Unix() - secs
		series, err := st.MetricSeries(targetID, metric, since)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, series)
	})

	mux.HandleFunc("/api/settings/background", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, Background())
		case http.MethodPost:
			var body struct {
				Enabled bool `json:"enabled"`
			}
			if !decodeJSON(w, r, &body) {
				return
			}
			if body.Enabled {
				if err := InstallBackground(configPath); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				if err := UninstallBackground(); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			writeJSON(w, Background())
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// --- Remote host proxy ---
	// GET/POST /api/remote/config — get or set remote host URL
	// GET /api/remote/* — proxy to host's API

	mux.HandleFunc("/api/remote/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			hostURL, adminToken := cs.RemoteConfig()
			writeJSON(w, map[string]string{"host_url": hostURL, "admin_token": adminToken})
		case http.MethodPost:
			var body struct {
				HostURL    string `json:"host_url"`
				AdminToken string `json:"admin_token"`
			}
			if !decodeJSON(w, r, &body) {
				return
			}
			hostURL := strings.TrimSpace(body.HostURL)
			adminToken := strings.TrimSpace(body.AdminToken)
			if hostURL != "" {
				normalized, err := normalizeRemoteHost(hostURL)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				hostURL = normalized
			}
			if err := cs.SetRemoteConfig(hostURL, adminToken); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]string{"host_url": hostURL, "admin_token": adminToken})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/remote/", func(w http.ResponseWriter, r *http.Request) {
		hostURL, adminToken := cs.RemoteConfig()
		if hostURL == "" {
			http.Error(w, "no remote host configured", http.StatusBadRequest)
			return
		}
		if _, err := normalizeRemoteHost(hostURL); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/remote")
		target := hostURL + "/api" + path
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}

		req, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
			return
		}
		if adminToken != "" {
			req.Header.Set("Authorization", "Bearer "+adminToken)
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	return mux
}

func normalizeRemoteHost(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("remote host must be an absolute http(s) URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("remote host must use http or https")
	}
	if u.User != nil {
		return "", fmt.Errorf("remote host must not include user info")
	}
	if strings.TrimSpace(u.RawQuery) != "" || strings.TrimSpace(u.Fragment) != "" {
		return "", fmt.Errorf("remote host must not include query or fragment")
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("remote host is missing host")
	}
	if ip := net.ParseIP(host); ip != nil && isBlockedRemoteIP(ip) {
		return "", fmt.Errorf("remote host IP is not allowed")
	}
	u.Path = strings.TrimRight(u.EscapedPath(), "/")
	return strings.TrimRight(u.String(), "/"), nil
}

func isBlockedRemoteIP(ip net.IP) bool {
	return ip.IsUnspecified() || ip.IsMulticast()
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
