package app

import "testing"

func TestLegacyTargetGetsDefaultMonitors(t *testing.T) {
	tgt := Target{
		ID:              "legacy",
		Name:            "Legacy",
		Host:            "mc.example.com",
		Port:            25565,
		IntervalSec:     45,
		TimeoutMs:       1200,
		ProbesPerBurst:  3,
		ProbeGapMs:      250,
		ProtocolVersion: 760,
	}.normalized()

	if !tgt.Monitors.Online.Enabled || !tgt.Monitors.Players.Enabled || !tgt.Monitors.Latency.Enabled || !tgt.Monitors.Loss.Enabled {
		t.Fatalf("legacy target monitors not all enabled: %#v", tgt.Monitors)
	}
	if tgt.Monitors.Online.IntervalSec != 45 || tgt.Monitors.Players.IntervalSec != 45 {
		t.Fatalf("online/players interval = %d/%d, want 45", tgt.Monitors.Online.IntervalSec, tgt.Monitors.Players.IntervalSec)
	}
	if tgt.Monitors.Latency.ProtocolVersion != 760 || tgt.Monitors.Latency.ProbesPerBurst != 3 || tgt.Monitors.Latency.ProbeGapMs != 250 {
		t.Fatalf("latency monitor did not inherit legacy probe settings: %#v", tgt.Monitors.Latency)
	}
	if tgt.Monitors.Loss.ProbesPerBurst != 3 || tgt.Monitors.Loss.ProbeGapMs != 250 {
		t.Fatalf("loss monitor did not inherit legacy burst settings: %#v", tgt.Monitors.Loss)
	}
}
