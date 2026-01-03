//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	serviceName = "com.gptwol.agent"
	plistFile   = "/Library/LaunchDaemons/com.gptwol.agent.plist"
	installPath = "/usr/local/bin/gptwol-agent"
)

func installService() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Copy binary to /usr/local/bin
	if exePath != installPath {
		input, err := os.ReadFile(exePath)
		if err != nil {
			return fmt.Errorf("failed to read binary: %v", err)
		}
		if err := os.WriteFile(installPath, input, 0755); err != nil {
			return fmt.Errorf("failed to copy binary to %s: %v", installPath, err)
		}
		fmt.Printf("Binary copied to %s\n", installPath)
	}

	// Build program arguments
	programArgs := fmt.Sprintf(`        <string>%s</string>
        <string>-action</string>
        <string>%s</string>`, installPath, action)

	if macAddress != "" {
		programArgs += fmt.Sprintf(`
        <string>-mac</string>
        <string>%s</string>`, macAddress)
	}

	// Create launchd plist
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
%s
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/gptwol-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/gptwol-agent.log</string>
</dict>
</plist>
`, serviceName, programArgs)

	if err := os.WriteFile(plistFile, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist file: %v", err)
	}

	// Load service
	cmd := exec.Command("launchctl", "load", plistFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: failed to load service: %v\n", err)
	}

	fmt.Printf("\nService installed successfully!\n")
	fmt.Printf("  Name: %s\n", serviceName)
	fmt.Printf("  Action: %s\n", action)
	fmt.Printf("  Binary: %s\n", installPath)
	fmt.Printf("  Plist: %s\n", plistFile)
	fmt.Printf("  Status: launchctl list | grep gptwol\n")
	fmt.Printf("  Logs: tail -f /var/log/gptwol-agent.log\n")

	return nil
}

func uninstallService() error {
	// Unload service
	cmd := exec.Command("launchctl", "unload", plistFile)
	cmd.Run()

	// Remove plist
	if err := os.Remove(plistFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to remove plist file: %v\n", err)
	}

	// Remove binary
	if err := os.Remove(installPath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to remove binary: %v\n", err)
	}

	fmt.Println("Service uninstalled successfully!")
	return nil
}
