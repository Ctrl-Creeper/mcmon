package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const serviceLabel = "com.lewiswu.mcmon"

// installService registers the current binary to run in the background and
// start automatically at login/boot, using the platform's native mechanism.
func installService(configPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	if configPath == "" {
		configPath = "config.json"
	}
	configPath, err = filepath.Abs(configPath)
	if err != nil {
		return err
	}
	wd := filepath.Dir(configPath)

	switch runtime.GOOS {
	case "darwin":
		return installLaunchd(exe, wd, configPath)
	case "linux":
		return installSystemd(exe, wd, configPath)
	case "windows":
		return installWindowsTask(exe, configPath)
	default:
		return fmt.Errorf("background install not supported on %s", runtime.GOOS)
	}
}

func uninstallService() error {
	switch runtime.GOOS {
	case "darwin":
		return uninstallLaunchd()
	case "linux":
		return uninstallSystemd()
	case "windows":
		return uninstallWindowsTask()
	default:
		return fmt.Errorf("background uninstall not supported on %s", runtime.GOOS)
	}
}

type BackgroundStatus struct {
	Platform  string `json:"platform"`
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Detail    string `json:"detail"`
}

func backgroundStatus() BackgroundStatus {
	status := BackgroundStatus{Platform: runtime.GOOS}
	switch runtime.GOOS {
	case "darwin":
		status.Supported = true
		path, err := launchdPlistPath()
		status.Enabled = err == nil && fileExists(path)
		status.Detail = path
	case "linux":
		status.Supported = true
		path, err := systemdUnitPath()
		status.Enabled = err == nil && fileExists(path)
		status.Detail = path
	case "windows":
		status.Supported = true
		status.Enabled = run("schtasks", "/Query", "/TN", windowsTaskName) == nil
		status.Detail = windowsTaskName
	default:
		status.Detail = "background mode is not supported on this platform"
	}
	return status
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// --- macOS: launchd user agent ---

func launchdPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", serviceLabel+".plist"), nil
}

func installLaunchd(exe, wd, configPath string) error {
	path, err := launchdPlistPath()
	if err != nil {
		return err
	}
	plist := launchdPlist(exe, wd, configPath)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(plist), 0644); err != nil {
		return err
	}
	if err := run("launchctl", "unload", path); err != nil {
		_ = err // ignore: may not be loaded yet
	}
	if err := run("launchctl", "load", path); err != nil {
		return err
	}
	fmt.Printf("Installed launchd agent at %s and started it.\n", path)
	return nil
}

func launchdPlist(exe, wd, configPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>serve</string>
    <string>-config</string>
    <string>%s</string>
  </array>
  <key>WorkingDirectory</key><string>%s</string>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>%s/mcmon.log</string>
  <key>StandardErrorPath</key><string>%s/mcmon.err.log</string>
</dict>
</plist>
`, serviceLabel, exe, configPath, wd, wd, wd)
}

func uninstallLaunchd() error {
	path, err := launchdPlistPath()
	if err != nil {
		return err
	}
	_ = run("launchctl", "unload", path)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	fmt.Println("Removed launchd agent.")
	return nil
}

// --- Linux: systemd user unit ---

func systemdUnitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user", "mcmon.service"), nil
}

func installSystemd(exe, wd, configPath string) error {
	path, err := systemdUnitPath()
	if err != nil {
		return err
	}
	unit := systemdUnit(exe, wd, configPath)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(unit), 0644); err != nil {
		return err
	}
	if err := run("systemctl", "--user", "daemon-reload"); err != nil {
		return err
	}
	if err := run("systemctl", "--user", "enable", "--now", "mcmon.service"); err != nil {
		return err
	}
	fmt.Printf("Installed systemd user unit at %s and started it.\n", path)
	fmt.Println("If this is a headless server, run: loginctl enable-linger $USER")
	return nil
}

func systemdUnit(exe, wd, configPath string) string {
	return fmt.Sprintf(`[Unit]
Description=Minecraft latency monitor

[Service]
ExecStart=%s serve -config %s
WorkingDirectory=%s
Restart=on-failure

[Install]
WantedBy=default.target
`, systemdQuote(exe), systemdQuote(configPath), systemdQuote(wd))
}

func uninstallSystemd() error {
	_ = run("systemctl", "--user", "disable", "--now", "mcmon.service")
	path, err := systemdUnitPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	_ = run("systemctl", "--user", "daemon-reload")
	fmt.Println("Removed systemd user unit.")
	return nil
}

// --- Windows: Scheduled Task running at logon ---

const windowsTaskName = "McLatencyMonitor"

func installWindowsTask(exe, configPath string) error {
	args := windowsTaskArgs(exe, configPath)
	if err := run("schtasks", args...); err != nil {
		return err
	}
	// Start it immediately too, instead of waiting for next logon.
	_ = run("schtasks", "/Run", "/TN", windowsTaskName)
	fmt.Printf("Installed scheduled task %q (runs at logon).\n", windowsTaskName)
	return nil
}

func windowsTaskArgs(exe, configPath string) []string {
	return []string{
		"/Create", "/F", "/SC", "ONLOGON",
		"/TN", windowsTaskName,
		"/TR", windowsTaskCommand(exe, configPath),
	}
}

func windowsTaskCommand(exe, configPath string) string {
	return fmt.Sprintf(`"%s" serve -config "%s"`, exe, configPath)
}

func systemdQuote(s string) string {
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func uninstallWindowsTask() error {
	if err := run("schtasks", "/Delete", "/F", "/TN", windowsTaskName); err != nil {
		return err
	}
	fmt.Printf("Removed scheduled task %q.\n", windowsTaskName)
	return nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
