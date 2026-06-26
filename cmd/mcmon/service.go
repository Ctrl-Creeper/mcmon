package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const serviceLabel = "com.lewiswu.mcmon"

// installService registers the current binary to run in the background and
// start automatically at login/boot, using the platform's native mechanism.
func installService() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "darwin":
		return installLaunchd(exe, wd)
	case "linux":
		return installSystemd(exe, wd)
	case "windows":
		return installWindowsTask(exe, wd)
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

// --- macOS: launchd user agent ---

func launchdPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", serviceLabel+".plist"), nil
}

func installLaunchd(exe, wd string) error {
	path, err := launchdPlistPath()
	if err != nil {
		return err
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
  </array>
  <key>WorkingDirectory</key><string>%s</string>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>%s/mcmon.log</string>
  <key>StandardErrorPath</key><string>%s/mcmon.err.log</string>
</dict>
</plist>
`, serviceLabel, exe, wd, wd, wd)

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

func installSystemd(exe, wd string) error {
	path, err := systemdUnitPath()
	if err != nil {
		return err
	}
	unit := fmt.Sprintf(`[Unit]
Description=Minecraft latency monitor

[Service]
ExecStart=%s
WorkingDirectory=%s
Restart=on-failure

[Install]
WantedBy=default.target
`, exe, wd)

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

func installWindowsTask(exe, wd string) error {
	// /SC ONLOGON keeps it running in the background after sign-in;
	// /RL HIGHEST avoids extra UAC prompts when started this way.
	args := []string{
		"/Create", "/F", "/SC", "ONLOGON", "/RL", "HIGHEST",
		"/TN", windowsTaskName,
		"/TR", fmt.Sprintf(`"%s" -config "%s"`, exe, filepath.Join(wd, "config.json")),
	}
	if err := run("schtasks", args...); err != nil {
		return err
	}
	// Start it immediately too, instead of waiting for next logon.
	_ = run("schtasks", "/Run", "/TN", windowsTaskName)
	fmt.Printf("Installed scheduled task %q (runs at logon).\n", windowsTaskName)
	return nil
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
