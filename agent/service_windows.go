//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const serviceName = "gptwol-agent"

func installService() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	// Get absolute path
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Build binPath
	binPath := fmt.Sprintf("\"%s\" -action \"%s\"", exePath, action)
	if macAddress != "" {
		binPath = fmt.Sprintf("%s -mac \"%s\"", binPath, macAddress)
	}

	// Create service using sc.exe (binPath= must be together with value)
	cmd := exec.Command("sc.exe", "create", serviceName,
		fmt.Sprintf("binPath=%s", binPath),
		"start=auto",
		fmt.Sprintf("DisplayName=%s", "GPTWol Agent"),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	// Set description
	cmd = exec.Command("sc.exe", "description", serviceName, "GPTWol Sleep-on-LAN agent - listens for magic packets")
	cmd.Run()

	// Start service
	cmd = exec.Command("sc.exe", "start", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: failed to start service: %v\n", err)
		fmt.Println("You can start it manually: sc start gptwol-agent")
	}

	fmt.Printf("\nService installed successfully!\n")
	fmt.Printf("  Name: %s\n", serviceName)
	fmt.Printf("  Action: %s\n", action)
	fmt.Printf("  Status: sc query %s\n", serviceName)

	return nil
}

func uninstallService() error {
	// Stop service first
	cmd := exec.Command("sc.exe", "stop", serviceName)
	cmd.Run() // Ignore error if not running

	// Delete service
	cmd = exec.Command("sc.exe", "delete", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete service: %v", err)
	}

	fmt.Println("Service uninstalled successfully!")
	return nil
}
