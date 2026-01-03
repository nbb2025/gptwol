package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var (
	version    = "1.0.0"
	macAddress string
	action     string
)

// Magic Packet: 6 bytes of 0xFF followed by 16 repetitions of target MAC
const magicPacketSize = 102

func getMACAddresses() ([]string, error) {
	var macs []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		// Skip loopback and interfaces without MAC
		if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
			continue
		}
		macs = append(macs, strings.ToLower(iface.HardwareAddr.String()))
	}
	return macs, nil
}

func reverseMac(mac string) string {
	parts := strings.Split(mac, ":")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ":")
}

func parseMagicPacket(data []byte) (string, bool) {
	if len(data) < magicPacketSize {
		return "", false
	}

	// Check for 6 bytes of 0xFF
	for i := 0; i < 6; i++ {
		if data[i] != 0xFF {
			return "", false
		}
	}

	// Extract MAC address (bytes 6-11)
	mac := data[6:12]

	// Verify 16 repetitions of MAC
	for i := 0; i < 16; i++ {
		offset := 6 + (i * 6)
		if !bytes.Equal(data[offset:offset+6], mac) {
			return "", false
		}
	}

	// Format MAC address
	macStr := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])

	return macStr, true
}

func executeAction(action string) {
	log.Printf("[INFO] Executing action: %s", action)

	time.Sleep(2 * time.Second)

	var cmd *exec.Cmd
	switch action {
	case "shutdown":
		if runtime.GOOS == "windows" {
			cmd = exec.Command("shutdown", "/s", "/t", "5")
		} else {
			cmd = exec.Command("shutdown", "-h", "now")
		}
	case "reboot":
		if runtime.GOOS == "windows" {
			cmd = exec.Command("shutdown", "/r", "/t", "5")
		} else {
			cmd = exec.Command("shutdown", "-r", "now")
		}
	case "sleep":
		if runtime.GOOS == "windows" {
			cmd = exec.Command("rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0")
		} else {
			cmd = exec.Command("systemctl", "suspend")
		}
	case "hibernate":
		if runtime.GOOS == "windows" {
			cmd = exec.Command("shutdown", "/h")
		} else {
			cmd = exec.Command("systemctl", "hibernate")
		}
	default:
		log.Printf("[ERROR] Unknown action: %s", action)
		return
	}

	if err := cmd.Run(); err != nil {
		log.Printf("[ERROR] Action failed: %v", err)
	}
}

func listenUDP(port int, targetMACs map[string]bool, reversedMACs map[string]bool) {
	addr := net.UDPAddr{
		Port: port,
		IP:   net.IPv4zero,
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("[ERROR] Failed to listen on UDP port %d: %v", port, err)
	}
	defer conn.Close()

	log.Printf("[INFO] Listening for Magic Packets on UDP port %d", port)

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("[ERROR] Read error: %v", err)
			continue
		}

		mac, valid := parseMagicPacket(buf[:n])
		if !valid {
			continue
		}

		mac = strings.ToLower(mac)
		log.Printf("[INFO] Received Magic Packet for MAC %s from %s", mac, remoteAddr)

		// Check if it's a reversed MAC (Sleep-on-LAN signal)
		if reversedMACs[mac] {
			log.Printf("[INFO] Sleep-on-LAN packet detected! Executing: %s", action)
			go executeAction(action)
		} else if targetMACs[mac] {
			log.Printf("[INFO] Wake-on-LAN packet for this machine (ignored - already awake)")
		}
	}
}

func main() {
	var showVersion bool
	var install bool
	var uninstall bool
	var port int
	var listMACs bool

	flag.StringVar(&macAddress, "mac", "", "MAC address to monitor (auto-detect if empty)")
	flag.StringVar(&action, "action", "shutdown", "Action on SOL packet: shutdown, reboot, sleep, hibernate")
	flag.IntVar(&port, "port", 9, "UDP port to listen on (default: 9)")
	flag.BoolVar(&listMACs, "list-macs", false, "List all MAC addresses and exit")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&install, "install", false, "Install as system service")
	flag.BoolVar(&uninstall, "uninstall", false, "Uninstall system service")
	flag.Parse()

	if showVersion {
		fmt.Printf("gptwol-agent %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	if listMACs {
		macs, err := getMACAddresses()
		if err != nil {
			log.Fatalf("Failed to get MAC addresses: %v", err)
		}
		fmt.Println("Available MAC addresses:")
		for _, mac := range macs {
			fmt.Printf("  %s (reversed: %s)\n", mac, reverseMac(mac))
		}
		os.Exit(0)
	}

	if install {
		if err := installService(); err != nil {
			log.Fatalf("Install failed: %v", err)
		}
		fmt.Println("Service installed successfully")
		os.Exit(0)
	}

	if uninstall {
		if err := uninstallService(); err != nil {
			log.Fatalf("Uninstall failed: %v", err)
		}
		fmt.Println("Service uninstalled successfully")
		os.Exit(0)
	}

	// Get MAC addresses to monitor
	var macs []string
	if macAddress != "" {
		macs = []string{strings.ToLower(macAddress)}
	} else {
		var err error
		macs, err = getMACAddresses()
		if err != nil {
			log.Fatalf("Failed to get MAC addresses: %v", err)
		}
	}

	if len(macs) == 0 {
		log.Fatal("No MAC addresses found")
	}

	// Build lookup maps
	targetMACs := make(map[string]bool)
	reversedMACs := make(map[string]bool)

	for _, mac := range macs {
		targetMACs[mac] = true
		reversed := reverseMac(mac)
		reversedMACs[reversed] = true
		log.Printf("[INFO] Monitoring MAC: %s (SOL trigger: %s)", mac, reversed)
	}

	log.Printf("[INFO] gptwol-agent %s starting", version)
	log.Printf("[INFO] Platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Printf("[INFO] Action on SOL: %s", action)

	// Listen on both common WOL ports
	go listenUDP(7, targetMACs, reversedMACs)
	go listenUDP(9, targetMACs, reversedMACs)

	// Also listen on custom port if different
	if port != 7 && port != 9 {
		go listenUDP(port, targetMACs, reversedMACs)
	}

	// Keep main goroutine alive
	select {}
}
