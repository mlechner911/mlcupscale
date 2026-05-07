// Copyright (c) 2026 Michael Lechner
// MIT License

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/getlantern/systray"

	"upscale-service/internal/osutil"
)

var (
	cmd     *exec.Cmd
	cmdMu   sync.Mutex
	running bool
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("MLC Upscale")
	systray.SetTooltip("MLC Upscale Server Control")

	// Set Icon
	exe, _ := os.Executable()
	iconPath := filepath.Join(filepath.Dir(exe), "icon.ico")
	iconData, err := os.ReadFile(iconPath)
	if err == nil {
		systray.SetIcon(iconData)
	}

	mStatus := systray.AddMenuItem("Status: Stopped", "Server Status")
	mStatus.Disable()

	systray.AddSeparator()
	mStart := systray.AddMenuItem("Start Server", "Start the background process")
	mStop := systray.AddMenuItem("Stop Server", "Kill the background process")
	mStop.Disable()

	systray.AddSeparator()
	mOpenWeb := systray.AddMenuItem("Open Web UI", "Open the server web interface")
	mOpenDocs := systray.AddMenuItem("Open API Docs", "Open Swagger documentation")

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit completely")

	// Initial check if already running (unlikely on startup but good for stability)
	updateStatus(mStatus, mStart, mStop)

	go func() {
		for {
			select {
			case <-mStart.ClickedCh:
				startServer(mStatus, mStart, mStop)
			case <-mStop.ClickedCh:
				stopServer(mStatus, mStart, mStop)
			case <-mOpenWeb.ClickedCh:
				openURL("http://127.0.0.1:8089/api/v1/health") // Placeholder or web UI if implemented
			case <-mOpenDocs.ClickedCh:
				openURL("http://127.0.0.1:8089/api/v1/docs")
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()

	// Watchdog to update status if process dies
	go func() {
		for {
			time.Sleep(5 * time.Second)
			updateStatus(mStatus, mStart, mStop)
		}
	}()
}

func onExit() {
	stopServer(nil, nil, nil)
}

func getExePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "upscale-server"
	}
	dir := filepath.Dir(exe)
	
	// Check for various possible names
	names := []string{"upscale-server.exe", "mlcupscale-bin", "upscale-server"}
	for _, name := range names {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "upscale-server"
}

func startServer(mStatus *systray.MenuItem, mStart *systray.MenuItem, mStop *systray.MenuItem) {
	cmdMu.Lock()
	defer cmdMu.Unlock()

	if running {
		return
	}

	serverPath := getExePath()
	// Pass explicit config if it exists in the same directory
	args := []string{}
	configPath := filepath.Join(filepath.Dir(serverPath), "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		args = append(args, "-config", configPath)
	} else {
		// Try config/config.yaml
		configPath = filepath.Join(filepath.Dir(serverPath), "config", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			args = append(args, "-config", configPath)
		}
	}

	cmd = execCommand(serverPath, args...)
	
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}

	running = true
	if mStatus != nil {
		mStatus.SetTitle("Status: Running")
		mStart.Disable()
		mStop.Enable()
	}
}

func stopServer(mStatus *systray.MenuItem, mStart *systray.MenuItem, mStop *systray.MenuItem) {
	cmdMu.Lock()
	defer cmdMu.Unlock()

	if !running {
		return
	}

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	
	// Fallback for Windows: Taskkill if process is orphaned or multiple instances exist
	if runtime.GOOS == "windows" {
		_ = execCommand("taskkill", "/F", "/IM", "upscale-server.exe", "/T").Run()
	}

	running = false
	if mStatus != nil {
		mStatus.SetTitle("Status: Stopped")
		mStart.Enable()
		mStop.Disable()
	}
}

func updateStatus(mStatus *systray.MenuItem, mStart *systray.MenuItem, mStop *systray.MenuItem) {
	cmdMu.Lock()
	defer cmdMu.Unlock()

	// Check if process is still alive
	isAlive := false
	if cmd != nil && cmd.Process != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			isAlive = false
		} else {
            // Signal(0) is not supported on Windows but we can check if it returns error
            if runtime.GOOS == "windows" {
                // Check via tasklist but with hidden window
                out, _ := execCommand("tasklist", "/NH", "/FI", "IMAGENAME eq upscale-server.exe").Output()
                if strings.Contains(string(out), "upscale-server.exe") {
                    isAlive = true
                }
            } else {
                if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
                    isAlive = true
                }
            }
		}
	} else {
        // Just check by name if we didn't start it
        if runtime.GOOS == "windows" {
            out, _ := execCommand("tasklist", "/NH", "/FI", "IMAGENAME eq upscale-server.exe").Output()
            if strings.Contains(string(out), "upscale-server.exe") {
                isAlive = true
            }
        }
    }

	if isAlive {
		running = true
		if mStatus != nil {
			mStatus.SetTitle("Status: Running")
			mStart.Disable()
			mStop.Enable()
		}
	} else {
		running = false
		if mStatus != nil {
			mStatus.SetTitle("Status: Stopped")
			mStart.Enable()
			mStop.Disable()
		}
	}
}

func openURL(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = execCommand("xdg-open", url).Start()
	case "windows":
		err = execCommand("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = execCommand("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Printf("Failed to open URL: %v\n", err)
	}
}

func execCommand(name string, arg ...string) *exec.Cmd {
	return osutil.PrepareCommand(exec.Command(name, arg...))
}

