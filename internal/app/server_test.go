package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ctrl-Creeper/mcmon/internal/store"
)

func newTestMux(t *testing.T, cfg Config) (*ConfigStore, *http.ServeMux) {
	t.Helper()
	cfgPath := t.TempDir() + "/config.json"
	if err := writeConfig(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	cs, err := openConfigStore(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(t.TempDir() + "/mcmon.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return cs, newMux(st, cs, NewManager(st), cfgPath)
}

func TestDefaultConfigBindsLocalhostForDesktopSafety(t *testing.T) {
	cfg := defaultConfig()
	if cfg.ListenAddr != "127.0.0.1:8090" {
		t.Fatalf("ListenAddr = %q, want localhost bind", cfg.ListenAddr)
	}
	if cfg.RemoteHost != "" || cfg.RemoteSessionToken != "" {
		t.Fatalf("remote config should default empty, got host=%q session=%q", cfg.RemoteHost, cfg.RemoteSessionToken)
	}
}

func TestRemoteConfigStoresHostAndUsernameOnly(t *testing.T) {
	_, mux := newTestMux(t, defaultConfig())
	body := strings.NewReader(`{"host_url":"http://host.example:9090","username":"admin"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/remote/config", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/remote/config = %d, want 200: %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/remote/config", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/remote/config = %d, want 200", rr.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["host_url"] != "http://host.example:9090" || got["username"] != "admin" || got["session_token"] != "" {
		t.Fatalf("remote config = %#v", got)
	}
}

func TestRemoteLoginStoresSessionAndProxyForwardsBearerToken(t *testing.T) {
	var gotAuth string
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/login" {
			if r.Method != http.MethodPost {
				t.Errorf("login method = %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"session_token":"session-secret","expires_at":9999999999,"username":"admin","totp_enabled":false}`))
			return
		}
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/api/agents" {
			t.Errorf("path = %q, want /api/agents", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer remote.Close()

	cfg := defaultConfig()
	_, mux := newTestMux(t, cfg)

	body := strings.NewReader(`{"host_url":"` + remote.URL + `","username":"admin","password":"secret-password"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/remote/login", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("remote login = %d, want 200: %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/remote/agents", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("remote proxy = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	if gotAuth != "Bearer session-secret" {
		t.Fatalf("Authorization = %q, want Bearer session-secret", gotAuth)
	}
}

func TestRemoteConfigRejectsUnsupportedURLScheme(t *testing.T) {
	_, mux := newTestMux(t, defaultConfig())
	body := strings.NewReader(`{"host_url":"file:///etc/passwd","username":"admin"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/remote/config", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unsupported scheme status = %d, want 400", rr.Code)
	}
}
