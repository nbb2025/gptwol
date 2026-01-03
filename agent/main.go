package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

var (
	version  = "1.0.0"
	password string
	port     string
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Action  string `json:"action,omitempty"`
}

func jsonResponse(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqPassword := r.URL.Query().Get("password")
		if reqPassword == "" {
			reqPassword = r.Header.Get("X-API-Password")
		}

		if reqPassword != password {
			log.Printf("[WARN] Unauthorized request from %s", r.RemoteAddr)
			jsonResponse(w, http.StatusUnauthorized, Response{
				Success: false,
				Message: "Invalid password",
			})
			return
		}
		next(w, r)
	}
}

func executeCommand(action string, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Shutdown request from %s", r.RemoteAddr)

	jsonResponse(w, http.StatusOK, Response{
		Success: true,
		Message: "Shutdown initiated",
		Action:  "shutdown",
	})

	go func() {
		time.Sleep(1 * time.Second)
		var err error
		if runtime.GOOS == "windows" {
			err = executeCommand("shutdown", "shutdown", "/s", "/t", "5")
		} else {
			err = executeCommand("shutdown", "shutdown", "-h", "now")
		}
		if err != nil {
			log.Printf("[ERROR] Shutdown failed: %v", err)
		}
	}()
}

func rebootHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Reboot request from %s", r.RemoteAddr)

	jsonResponse(w, http.StatusOK, Response{
		Success: true,
		Message: "Reboot initiated",
		Action:  "reboot",
	})

	go func() {
		time.Sleep(1 * time.Second)
		var err error
		if runtime.GOOS == "windows" {
			err = executeCommand("reboot", "shutdown", "/r", "/t", "5")
		} else {
			err = executeCommand("reboot", "shutdown", "-r", "now")
		}
		if err != nil {
			log.Printf("[ERROR] Reboot failed: %v", err)
		}
	}()
}

func sleepHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Sleep request from %s", r.RemoteAddr)

	jsonResponse(w, http.StatusOK, Response{
		Success: true,
		Message: "Sleep initiated",
		Action:  "sleep",
	})

	go func() {
		time.Sleep(1 * time.Second)
		var err error
		if runtime.GOOS == "windows" {
			// Windows sleep using rundll32
			err = executeCommand("sleep", "rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0")
		} else {
			// Linux sleep using systemctl
			err = executeCommand("sleep", "systemctl", "suspend")
		}
		if err != nil {
			log.Printf("[ERROR] Sleep failed: %v", err)
		}
	}()
}

func hibernateHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Hibernate request from %s", r.RemoteAddr)

	jsonResponse(w, http.StatusOK, Response{
		Success: true,
		Message: "Hibernate initiated",
		Action:  "hibernate",
	})

	go func() {
		time.Sleep(1 * time.Second)
		var err error
		if runtime.GOOS == "windows" {
			err = executeCommand("hibernate", "shutdown", "/h")
		} else {
			err = executeCommand("hibernate", "systemctl", "hibernate")
		}
		if err != nil {
			log.Printf("[ERROR] Hibernate failed: %v", err)
		}
	}()
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, Response{
		Success: true,
		Message: fmt.Sprintf("gptwol-agent %s running on %s/%s", version, runtime.GOOS, runtime.GOARCH),
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	var showVersion bool
	var install bool
	var uninstall bool

	flag.StringVar(&password, "password", "", "API password (required, or set AGENT_PASSWORD env)")
	flag.StringVar(&port, "port", "9009", "Listen port (default: 9009)")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&install, "install", false, "Install as system service")
	flag.BoolVar(&uninstall, "uninstall", false, "Uninstall system service")
	flag.Parse()

	if showVersion {
		fmt.Printf("gptwol-agent %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
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

	// Get password from env if not provided
	if password == "" {
		password = os.Getenv("AGENT_PASSWORD")
	}
	if password == "" {
		log.Fatal("Password required: use -password flag or set AGENT_PASSWORD environment variable")
	}

	// Get port from env if not provided via flag
	if envPort := os.Getenv("AGENT_PORT"); envPort != "" && port == "9009" {
		port = envPort
	}

	// Setup routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/status", authMiddleware(statusHandler))
	http.HandleFunc("/shutdown", authMiddleware(shutdownHandler))
	http.HandleFunc("/reboot", authMiddleware(rebootHandler))
	http.HandleFunc("/sleep", authMiddleware(sleepHandler))
	http.HandleFunc("/hibernate", authMiddleware(hibernateHandler))

	addr := fmt.Sprintf(":%s", port)
	log.Printf("[INFO] gptwol-agent %s starting on %s", version, addr)
	log.Printf("[INFO] Platform: %s/%s", runtime.GOOS, runtime.GOARCH)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
