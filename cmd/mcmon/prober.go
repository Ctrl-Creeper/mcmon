package main

import (
	"log"
	"sort"
	"time"

	"github.com/lewiswu/mc-latency-monitor/internal/mcping"
	"github.com/lewiswu/mc-latency-monitor/internal/store"
)

// runProbeOnce probes a target N times and writes one aggregated sample.
func runProbeOnce(st *store.Store, t Target) {
	n := t.ProbesPerMinute
	if n <= 0 {
		n = 5
	}
	timeout := time.Duration(t.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	gap := time.Duration(t.ProbeIntervalMs) * time.Millisecond
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

// startProbeLoop runs runProbeOnce for every target once per minute.
func startProbeLoop(st *store.Store, targets []Target) {
	probeAll := func() {
		for _, t := range targets {
			go runProbeOnce(st, t)
		}
	}
	probeAll()
	ticker := time.NewTicker(time.Minute)
	go func() {
		for range ticker.C {
			probeAll()
		}
	}()
}
