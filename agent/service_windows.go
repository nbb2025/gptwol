//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const serviceName = "GPTWol-Agent"

func installService() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Build arguments
	args := fmt.Sprintf("-action %s", action)
	if macAddress != "" {
		args = fmt.Sprintf("%s -mac %s", args, macAddress)
	}

	// Remove existing task if exists
	cmd := exec.Command("schtasks.exe", "/Delete", "/TN", serviceName, "/F")
	cmd.Run() // Ignore error if not exists

	// Create scheduled task that runs at startup
	cmd = exec.Command("schtasks.exe", "/Create",
		"/TN", serviceName,
		"/TR", fmt.Sprintf("\"%s\" %s", exePath, args),
		"/SC", "ONSTART",
		"/RU", "SYSTEM",
		"/RL", "HIGHEST",
		"/F",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create scheduled task: %v", err)
	}

	// Start the task immediately
	cmd = exec.Command("schtasks.exe", "/Run", "/TN", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: failed to start task: %v\n", err)
		fmt.Println("It will start automatically on next boot.")
	}

	fmt.Printf("\nScheduled task installed successfully!\n")
	fmt.Printf("  Name: %s\n", serviceName)
	fmt.Printf("  Action: %s\n", action)
	fmt.Printf("  Binary: %s\n", exePath)
	fmt.Printf("  Status: schtasks /Query /TN %s\n", serviceName)
	fmt.Printf("  Check process: tasklist /FI \"IMAGENAME eq gptwol-agent.exe\"\n")

	return nil
}

func uninstallService() error {
	// Kill running process
	cmd := exec.Command("taskkill.exe", "/F", "/IM", "gptwol-agent.exe")
	cmd.Run() // Ignore error if not running

	// Delete scheduled task
	cmd = exec.Command("schtasks.exe", "/Delete", "/TN", serviceName, "/F")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete scheduled task: %v", err)
	}

	fmt.Println("Scheduled task uninstalled successfully!")
	return nil
}
