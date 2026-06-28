package app

import (
	"encoding/json"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/lewiswu/mc-latency-monitor/internal/mcping"
	"github.com/lewiswu/mc-latency-monitor/internal/store"
)

const (
	MetricOnline  = "online"
	MetricPlayers = "players"
	MetricLatency = "latency"
	MetricLoss    = "loss"
)

type Manager struct {
	mu    sync.Mutex
	st    *store.Store
	stops map[string][]chan struct{}
}

func NewManager(st *store.Store) *Manager {
	return &Manager{st: st, stops: make(map[string][]chan struct{})}
}

func (m *Manager) Sync(targets []Target) {
	for _, t := range targets {
		m.Start(t)
	}
}

func (m *Manager) Start(t Target) {
	m.Stop(t.ID)
	t = t.normalized()

	var stops []chan struct{}
	add := func(metric string, interval int, fn func()) {
		stop := make(chan struct{})
		stops = append(stops, stop)
		go monitorLoop(t.ID, metric, interval, stop, fn)
	}

	if t.Monitors.Online.Enabled {
		add(MetricOnline, t.Monitors.Online.IntervalSec, func() { runOnlineOnce(m.st, t) })
	}
	if t.Monitors.Players.Enabled {
		add(MetricPlayers, t.Monitors.Players.IntervalSec, func() { runPlayersOnce(m.st, t) })
	}
	if t.Monitors.Latency.Enabled {
		add(MetricLatency, t.Monitors.Latency.IntervalSec, func() { runLatencyOnce(m.st, t) })
	}
	if t.Monitors.Loss.Enabled {
		add(MetricLoss, t.Monitors.Loss.IntervalSec, func() { runLossOnce(m.st, t) })
	}

	m.mu.Lock()
	m.stops[t.ID] = stops
	m.mu.Unlock()
}

func (m *Manager) Stop(id string) {
	m.mu.Lock()
	stops := m.stops[id]
	delete(m.stops, id)
	m.mu.Unlock()
	for _, stop := range stops {
		close(stop)
	}
}

func monitorLoop(targetID, metric string, intervalSec int, stop chan struct{}, fn func()) {
	fn()
	interval := time.Duration(intervalSec) * time.Second
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
			fn()
		}
	}
}

func runOnlineOnce(st *store.Store, t Target) {
	timeout := timeoutFor(t)
	res := mcping.StatusRequest(t.Host, t.Port, timeout, defaultProtocolVersion)
	value := 0.0
	if res.OK {
		value = 1
	}
	if err := st.InsertMetric(store.MetricSample{TargetID: t.ID, Metric: MetricOnline, Ts: time.Now().Unix(), Value: &value}); err != nil {
		log.Printf("store online failed for %s: %v", t.ID, err)
	}
}

func runPlayersOnce(st *store.Store, t Target) {
	timeout := timeoutFor(t)
	res := mcping.StatusRequest(t.Host, t.Port, timeout, defaultProtocolVersion)
	ts := time.Now().Unix()
	if !res.OK || res.PlayersOnline == nil {
		if err := st.InsertMetric(store.MetricSample{TargetID: t.ID, Metric: MetricPlayers, Ts: ts}); err != nil {
			log.Printf("store players failed for %s: %v", t.ID, err)
		}
		return
	}
	value := float64(*res.PlayersOnline)
	extra := ""
	if res.PlayersMax != nil {
		b, _ := json.Marshal(map[string]int{"max": *res.PlayersMax})
		extra = string(b)
	}
	if err := st.InsertMetric(store.MetricSample{TargetID: t.ID, Metric: MetricPlayers, Ts: ts, Value: &value, Extra: extra}); err != nil {
		log.Printf("store players failed for %s: %v", t.ID, err)
	}
}

func runLatencyOnce(st *store.Store, t Target) {
	result := runProbeBurst(t, t.Monitors.Latency)
	ts := time.Now().Unix()
	insertLegacySample(st, t.ID, ts, result)

	extra := latencyExtra(result)
	var value *float64
	if result.P50 != nil {
		value = result.P50
	}
	if err := st.InsertMetric(store.MetricSample{TargetID: t.ID, Metric: MetricLatency, Ts: ts, Value: value, Extra: extra}); err != nil {
		log.Printf("store latency failed for %s: %v", t.ID, err)
	}
}

func runLossOnce(st *store.Store, t Target) {
	result := runProbeBurst(t, t.Monitors.Loss)
	ts := time.Now().Unix()
	value := result.LossPct
	if err := st.InsertMetric(store.MetricSample{TargetID: t.ID, Metric: MetricLoss, Ts: ts, Value: &value}); err != nil {
		log.Printf("store loss failed for %s: %v", t.ID, err)
	}
}

type probeBurstResult struct {
	Min     *float64
	P50     *float64
	Max     *float64
	LossPct float64
}

func runProbeBurst(t Target, mon ProbeMonitor) probeBurstResult {
	n := mon.ProbesPerBurst
	if n <= 0 {
		n = 5
	}
	timeout := timeoutFor(t)
	gap := time.Duration(mon.ProbeGapMs) * time.Millisecond
	proto := mon.ProtocolVersion
	if proto == 0 {
		proto = defaultProtocolVersion
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

	out := probeBurstResult{LossPct: float64(fail) / float64(n)}
	if len(vals) > 0 {
		sort.Float64s(vals)
		min := vals[0]
		max := vals[len(vals)-1]
		p50 := vals[(len(vals)-1)/2]
		out.Min = &min
		out.P50 = &p50
		out.Max = &max
	}
	return out
}

func timeoutFor(t Target) time.Duration {
	timeout := time.Duration(t.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		return 1500 * time.Millisecond
	}
	return timeout
}

func insertLegacySample(st *store.Store, targetID string, ts int64, result probeBurstResult) {
	sample := store.Sample{TargetID: targetID, Ts: ts, Min: result.Min, P50: result.P50, Max: result.Max, LossPct: result.LossPct}
	if err := st.Insert(sample); err != nil {
		log.Printf("store legacy sample failed for %s: %v", targetID, err)
	}
}

func latencyExtra(result probeBurstResult) string {
	b, _ := json.Marshal(map[string]any{
		"min":      result.Min,
		"p50":      result.P50,
		"max":      result.Max,
		"loss_pct": result.LossPct,
	})
	return string(b)
}
