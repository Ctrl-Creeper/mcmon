package app

import (
	"strings"
	"testing"
)

func TestLaunchdPlistRunsLightweightServeMode(t *testing.T) {
	plist := launchdPlist("/Applications/MC Latency Monitor.app/Contents/MacOS/MC Latency Monitor", "/tmp/mc data", "/tmp/mc data/config.json")

	for _, want := range []string{
		"<string>/Applications/MC Latency Monitor.app/Contents/MacOS/MC Latency Monitor</string>",
		"<string>serve</string>",
		"<string>-config</string>",
		"<string>/tmp/mc data/config.json</string>",
		"<key>WorkingDirectory</key><string>/tmp/mc data</string>",
	} {
		if !strings.Contains(plist, want) {
			t.Fatalf("launchd plist missing %q:\n%s", want, plist)
		}
	}
}

func TestSystemdUnitRunsLightweightServeModeWithQuotedPaths(t *testing.T) {
	unit := systemdUnit("/opt/MC Latency Monitor/mcmon", "/tmp/mc data", "/tmp/mc data/config.json")

	for _, want := range []string{
		`ExecStart="/opt/MC Latency Monitor/mcmon" serve -config "/tmp/mc data/config.json"`,
		`WorkingDirectory="/tmp/mc data"`,
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("systemd unit missing %q:\n%s", want, unit)
		}
	}
}

func TestWindowsTaskCommandRunsLightweightServeMode(t *testing.T) {
	got := windowsTaskCommand(`C:\Program Files\MC Latency Monitor\mcmon.exe`, `C:\Users\Lewis\AppData\Roaming\mc-latency-monitor\config.json`)
	want := `"C:\Program Files\MC Latency Monitor\mcmon.exe" serve -config "C:\Users\Lewis\AppData\Roaming\mc-latency-monitor\config.json"`
	if got != want {
		t.Fatalf("windows task command = %q, want %q", got, want)
	}
}

func TestWindowsTaskArgsDoNotRequireAdministrator(t *testing.T) {
	args := windowsTaskArgs(`C:\Apps\mcmon.exe`, `C:\Users\Lewis\AppData\Roaming\mc-latency-monitor\config.json`)
	for _, arg := range args {
		if arg == "/RL" || arg == "HIGHEST" {
			t.Fatalf("scheduled task args should not request highest privileges: %#v", args)
		}
	}
}
