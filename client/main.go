package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/joho/godotenv"
	gonet "github.com/shirou/gopsutil/v3/net"
)

//go:embed manitor-logo.ico
var trayIcon []byte

const (
	collectInterval  = 1 * time.Second
	realtimeInterval = 1 * time.Second
	requestTimeout   = 30 * time.Second
	maxRetries       = 3
)

type Direction string

const (
	DirectionUpload   Direction = "upload"
	DirectionDownload Direction = "download"
)

type TrafficLog struct {
	RequestURL string    `json:"request_url"`
	PacketSize uint64    `json:"packet_size"`
	Direction  Direction `json:"direction"`
	Timestamp  time.Time `json:"timestamp"`
}

type AgentPayload struct {
	SystemIP  string       `json:"system_ip"`
	WiFiName  string       `json:"wifi_name"`
	Logs      []TrafficLog `json:"logs"`
	Collected time.Time    `json:"collected_at"`
}

type Agent struct {
	mu         sync.Mutex
	active     bool
	lastIO     []gonet.IOCountersStat
	httpClient *http.Client
	stopCh     chan struct{}
}

var (
	realtimePrevUp      uint64
	realtimePrevDown    uint64
	realtimeInitialized bool
	lastWiFiName        string
	lastWiFiCheck       time.Time
	serverClientURL     = "http://localhost:5000/api/v1/connections"
)

func NewAgent() *Agent {
	return &Agent{
		httpClient: &http.Client{Timeout: requestTimeout},
		stopCh:     make(chan struct{}),
	}
}

func (a *Agent) Activate() {
	a.mu.Lock()
	if a.active {
		a.mu.Unlock()
		return
	}
	a.active = true
	a.mu.Unlock()

	go a.run()
	log.Println("agent activated")
}

func (a *Agent) run() {
	ticker := time.NewTicker(collectInterval)
	defer ticker.Stop()

	a.collectAndSend()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.collectAndSend()
		}
	}
}

func (a *Agent) Shutdown() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.active {
		return
	}
	a.active = false
	close(a.stopCh)
	log.Println("agent deactivated")
}

func (a *Agent) collectAndSend() {
	payload, err := a.collectPayload()
	if err != nil {
		log.Printf("collect failed: %v", err)
		return
	}
	if len(payload.Logs) == 0 {
		log.Println("no telemetry to send")
		return
	}

	if err := a.sendWithRetry(payload); err != nil {
		log.Printf("send failed after %d retries: %v", maxRetries, err)
	}
}

func (a *Agent) collectPayload() (AgentPayload, error) {
	now := time.Now().UTC()
	systemIP := primaryIPv4()
	logs := make([]TrafficLog, 0, 2)
	wifiName := currentWiFiName(now)

	ioCounters, err := gonet.IOCounters(true)
	if err != nil {
		return AgentPayload{}, err
	}

	a.mu.Lock()
	prev := a.lastIO
	a.lastIO = ioCounters
	a.mu.Unlock()

	up, down := computeIODeltas(prev, ioCounters)
	if up > 0 {
		logs = append(logs, TrafficLog{
			RequestURL: "system://all-interfaces",
			PacketSize: up,
			Direction:  DirectionUpload,
			Timestamp:  now,
		})
	}
	if down > 0 {
		logs = append(logs, TrafficLog{
			RequestURL: "system://all-interfaces",
			PacketSize: down,
			Direction:  DirectionDownload,
			Timestamp:  now,
		})
	}

	return AgentPayload{
		SystemIP:  systemIP,
		WiFiName:  wifiName,
		Logs:      logs,
		Collected: now,
	}, nil
}

func computeIODeltas(prev, curr []gonet.IOCountersStat) (upload uint64, download uint64) {
	prevMap := make(map[string]gonet.IOCountersStat, len(prev))
	for _, p := range prev {
		prevMap[p.Name] = p
	}

	for _, c := range curr {
		p, ok := prevMap[c.Name]
		if !ok {
			continue
		}
		if c.BytesSent >= p.BytesSent {
			upload += c.BytesSent - p.BytesSent
		}
		if c.BytesRecv >= p.BytesRecv {
			download += c.BytesRecv - p.BytesRecv
		}
	}
	return upload, download
}

func primaryIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return ""
}

func (a *Agent) sendWithRetry(payload AgentPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var lastErr error
	for i := 1; i <= maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverClientURL, bytes.NewReader(body))
		if err != nil {
			cancel()
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.httpClient.Do(req)
		cancel()
		if err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_ = resp.Body.Close()
			log.Println("payload sent successfully")
			return nil
		}
		if resp != nil {
			if err == nil {
				err = errors.New(fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
			}
			_ = resp.Body.Close()
		}
		lastErr = err
		log.Printf("attempt %d failed: %v", i, err)
	}
	return lastErr
}

func onReady() {
	systray.SetIcon(trayIcon)
	systray.SetTitle("Manitor Client")
	systray.SetTooltip("Manitor Client Agent")

	activate := systray.AddMenuItem("Activate", "Activate client agent")
	exit := systray.AddMenuItem("Exit", "Deactivate and exit")

	agent := NewAgent()
	realtimeStop := make(chan struct{})
	realtimeStarted := false

	go func() {
		for {
			select {
			case <-activate.ClickedCh:
				agent.Activate()
				if !realtimeStarted {
					realtimeStarted = true
					go runRealtimeDebugLogs(realtimeStop)
				}
			case <-exit.ClickedCh:
				close(realtimeStop)
				agent.Shutdown()
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {}

func runRealtimeDebugLogs(stop <-chan struct{}) {
	ticker := time.NewTicker(realtimeInterval)
	defer ticker.Stop()

	log.Println("realtime debug logger started")
	logRealtimeSnapshot()

	for {
		select {
		case <-stop:
			log.Println("realtime debug logger stopped")
			return
		case <-ticker.C:
			logRealtimeSnapshot()
		}
	}
}

func logRealtimeSnapshot() {
	ioCounters, err := gonet.IOCounters(false)
	if err != nil {
		log.Printf("[realtime] i/o counters error: %v", err)
		return
	}

	var totalUp uint64
	var totalDown uint64
	for _, c := range ioCounters {
		totalUp += c.BytesSent
		totalDown += c.BytesRecv
	}

	upDelta := uint64(0)
	downDelta := uint64(0)
	if realtimeInitialized {
		if totalUp >= realtimePrevUp {
			upDelta = totalUp - realtimePrevUp
		}
		if totalDown >= realtimePrevDown {
			downDelta = totalDown - realtimePrevDown
		}
	}
	realtimePrevUp = totalUp
	realtimePrevDown = totalDown
	realtimeInitialized = true
	now := time.Now()
	wifiName := currentWiFiName(now)

	log.Printf(
		"[realtime] time=%s host_ip=%s wifi_name=%q upload_1s=%dB download_1s=%dB",
		now.Format(time.RFC3339),
		primaryIPv4(),
		wifiName,
		upDelta,
		downDelta,
	)
}

func currentWiFiName(now time.Time) string {
	if now.Sub(lastWiFiCheck) < 10*time.Second && lastWiFiName != "" {
		return lastWiFiName
	}

	cmd := exec.Command("netsh", "wlan", "show", "interfaces")
	out, err := cmd.Output()
	lastWiFiCheck = now
	if err != nil {
		lastWiFiName = "unknown"
		return lastWiFiName
	}

	lastWiFiName = parseSSID(string(out))
	if lastWiFiName == "" {
		lastWiFiName = "unknown"
	}
	return lastWiFiName
}

func parseSSID(raw string) string {
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Exclude BSSID and capture only the connected Wi-Fi name.
		if strings.HasPrefix(trimmed, "SSID") && !strings.HasPrefix(trimmed, "BSSID") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func main() {
	loadConfig()
	systray.Run(onReady, onExit)
}

func loadConfig() {
	_ = godotenv.Load()
	if v := strings.TrimSpace(os.Getenv("SERVER_CLIENT")); v != "" {
		serverClientURL = v
	}
	log.Printf("SERVER_CLIENT=%s", serverClientURL)
}
