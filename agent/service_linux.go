//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	serviceName    = "gptwol-agent"
	serviceFile    = "/etc/systemd/system/gptwol-agent.service"
	installPath    = "/usr/local/bin/gptwol-agent"
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

	if password == "" {
		return fmt.Errorf("password required for installation: use -password flag")
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

	// Create systemd service file
	serviceContent := fmt.Sprintf(`[Unit]
Description=GPTWol Agent - Remote shutdown/reboot service
After=network.target

[Service]
Type=simple
ExecStart=%s -password "%s" -port "%s"
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`, installPath, password, port)

	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %v", err)
	}

	// Reload systemd
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %v", err)
	}

	// Enable service
	cmd = exec.Command("systemctl", "enable", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %v", err)
	}

	// Start service
	cmd = exec.Command("systemctl", "start", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: failed to start service: %v\n", err)
		fmt.Println("You can start it manually: systemctl start gptwol-agent")
	}

	fmt.Printf("\nService installed successfully!\n")
	fmt.Printf("  Name: %s\n", serviceName)
	fmt.Printf("  Port: %s\n", port)
	fmt.Printf("  Binary: %s\n", installPath)
	fmt.Printf("  Service: %s\n", serviceFile)
	fmt.Printf("  Status: systemctl status %s\n", serviceName)
	fmt.Printf("  Logs: journalctl -u %s -f\n", serviceName)

	return nil
}

func uninstallService() error {
	// Stop service
	cmd := exec.Command("systemctl", "stop", serviceName)
	cmd.Run() // Ignore error if not running

	// Disable service
	cmd = exec.Command("systemctl", "disable", serviceName)
	cmd.Run()

	// Remove service file
	if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to remove service file: %v\n", err)
	}

	// Remove binary
	if err := os.Remove(installPath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to remove binary: %v\n", err)
	}

	// Reload systemd
	cmd = exec.Command("systemctl", "daemon-reload")
	cmd.Run()

	fmt.Println("Service uninstalled successfully!")
	return nil
}
