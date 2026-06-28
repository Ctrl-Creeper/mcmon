package store

import "testing"

func TestMetricSamplesRoundTrip(t *testing.T) {
	st, err := Open(t.TempDir() + "/mcmon.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	err = st.InsertMetric(MetricSample{
		TargetID: "srv",
		Metric:   "players",
		Ts:       123,
		Value:    ptr(12),
		Extra:    `{"max":40}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := st.MetricSeries("srv", "players", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].TargetID != "srv" || got[0].Metric != "players" || got[0].Ts != 123 || got[0].Value == nil || *got[0].Value != 12 || got[0].Extra != `{"max":40}` {
		t.Fatalf("metric sample = %#v", got[0])
	}
}

func ptr(v float64) *float64 { return &v }
