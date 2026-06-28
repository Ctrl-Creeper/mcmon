package mcping

import "testing"

func TestParseStatusPlayers(t *testing.T) {
	status, err := parseStatusJSON([]byte(`{"players":{"online":12,"max":40},"version":{"protocol":760}}`))
	if err != nil {
		t.Fatal(err)
	}
	if status.PlayersOnline == nil || *status.PlayersOnline != 12 {
		t.Fatalf("PlayersOnline = %#v, want 12", status.PlayersOnline)
	}
	if status.PlayersMax == nil || *status.PlayersMax != 40 {
		t.Fatalf("PlayersMax = %#v, want 40", status.PlayersMax)
	}
}
