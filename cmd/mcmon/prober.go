package main

import (
	"log"
	"sort"
	"sync"
	"time"

	"github.com/lewiswu/mc-latency-monitor/internal/mcping"
	"github.com/lewiswu/mc-latency-monitor/internal/store"
)

// Manager runs one independent probe loop per target, so each server can
// have its own polling interval and loops can be started/stopped/restarted
// individually when targets are added, edited, or removed via the API.
type Manager struct {
	mu    sync.Mutex
	st    *store.Store
	stops map[string]chan struct{}
}

func NewManager(st *store.Store) *Manager {
	return &Manager{st: st, stops: make(map[string]chan struct{})}
}

// Sync starts loops for all given targets, stopping any loop for a target
// that's no longer present. Call once at startup.
func (m *Manager) Sync(targets []Target) {
	for _, t := range targets {
		m.Start(t)
	}
}

// Start launches (or restarts, if already running) the probe loop for t.
func (m *Manager) Start(t Target) {
	m.Stop(t.ID)

	stop := make(chan struct{})
	m.mu.Lock()
	m.stops[t.ID] = stop
	m.mu.Unlock()

	go m.loop(t, stop)
}

// Stop halts the probe loop for the given target id, if running.
func (m *Manager) Stop(id string) {
	m.mu.Lock()
	stop, ok := m.stops[id]
	if ok {
		delete(m.stops, id)
	}
	m.mu.Unlock()
	if ok {
		close(stop)
	}
}

func (m *Manager) loop(t Target, stop chan struct{}) {
	runProbeOnce(m.st, t)

	interval := time.Duration(t.IntervalSec) * time.Second
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			runProbeOnce(m.st, t)
		}
	}
}

// runProbeOnce probes a target several times (a "burst") and writes one
// aggregated sample with min/median/max latency and packet loss.
func runProbeOnce(st *store.Store, t Target) {
	n := t.ProbesPerBurst
	if n <= 0 {
		n = 5
	}
	timeout := time.Duration(t.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	gap := time.Duration(t.ProbeGapMs) * time.Millisecond
	proto := t.ProtocolVersion
	if proto == 0 {
		proto = 760
	}

	var vals []float64
	fail := 0
	for i := 0; i < n; i++ {
		res := mcping.Ping(t.Host, t.Port, timeout, proto)
		if res.OK {
			vals = append(vals, res.LatencyMs)
		} else {
			fail++
		}
		if i < n-1 && gap > 0 {
			time.Sleep(gap)
		}
	}

	loss := float64(fail) / float64(n)
	sample := store.Sample{TargetID: t.ID, Ts: time.Now().Unix(), LossPct: loss}
	if len(vals) > 0 {
		sort.Float64s(vals)
		min := vals[0]
		max := vals[len(vals)-1]
		p50 := vals[(len(vals)-1)/2]
		sample.Min = &min
		sample.P50 = &p50
		sample.Max = &max
	}

	if err := st.Insert(sample); err != nil {
		log.Printf("store insert failed for %s: %v", t.ID, err)
		return
	}

	if len(vals) > 0 {
		log.Printf("%s p50=%.1fms loss=%.0f%%", t.ID, *sample.P50, loss*100)
	} else {
		log.Printf("%s unreachable loss=%.0f%%", t.ID, loss*100)
	}
}
