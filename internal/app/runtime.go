package app

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/Ctrl-Creeper/mcmon/internal/store"
)

type Runtime struct {
	ConfigPath string
	Config     *ConfigStore
	Store      *store.Store
	Manager    *Manager
	Handler    http.Handler
}

func StaticFS() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}

func NewRuntime(configPath string) (*Runtime, error) {
	cs, err := openConfigStore(configPath)
	if err != nil {
		return nil, err
	}
	st, err := store.Open(cs.Snapshot().DBPath)
	if err != nil {
		return nil, err
	}
	mgr := NewManager(st)
	mgr.Sync(cs.Targets())
	return &Runtime{
		ConfigPath: configPath,
		Config:     cs,
		Store:      st,
		Manager:    mgr,
		Handler:    newMux(st, cs, mgr, configPath),
	}, nil
}

func (r *Runtime) Close() error {
	if r.Store != nil {
		return r.Store.Close()
	}
	return nil
}

func RunServer(ctx context.Context, configPath string) error {
	rt, err := NewRuntime(configPath)
	if err != nil {
		return err
	}
	defer rt.Close()

	addr := rt.Config.Snapshot().ListenAddr
	srv := &http.Server{
		Addr:              addr,
		Handler:           rt.Handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	log.Printf("mcmon listening on %s", addr)
	err = srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func InstallBackground(configPath string) error {
	return installService(configPath)
}

func UninstallBackground() error {
	return uninstallService()
}

func Background() BackgroundStatus {
	return backgroundStatus()
}
