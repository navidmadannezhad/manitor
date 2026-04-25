package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

const (
	defaultDB   = "manitor.db"
	defaultAddr = ":5000"
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
	WiFiName  string       `json:"wifi_name,omitempty"`
	Collected time.Time    `json:"collected_at"`
	Logs      []TrafficLog `json:"logs"`
}

type Connection struct {
	ID            int64     `json:"id"`
	IP            string    `json:"ip"`
	WiFiName      string    `json:"wifi_name"`
	DownloadSize  uint64    `json:"download_size"`
	UploadSize    uint64    `json:"upload_size"`
	TotalDownload uint64    `json:"total_download"`
	TotalUpload   uint64    `json:"total_upload"`
	CreatedAt     time.Time `json:"created_at"`
}

type Server struct {
	db *sql.DB
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	listenAddr := loadConfig()

	db, err := openDB(defaultDB)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := ensureSchema(db); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	s := &Server{db: db}
	go s.runMidnightReset()

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIngest)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/v1/connections", s.handleConnections)
	mux.HandleFunc("/api/v1/connections/", s.handleConnectionByIPSocket)

	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("server listening on %s", listenAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen failed: %v", err)
		}
	}()

	waitForShutdown(httpServer)
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func ensureSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS connections (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ip TEXT NOT NULL,
	wifiname TEXT NOT NULL,
	download_size INTEGER NOT NULL,
	upload_size INTEGER NOT NULL,
	total_download INTEGER NOT NULL,
	total_upload INTEGER NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_connections_ip ON connections(ip);
CREATE INDEX IF NOT EXISTS idx_connections_created_at ON connections(created_at);
`)
	return err
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload AgentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.SystemIP) == "" {
		http.Error(w, "system_ip is required", http.StatusBadRequest)
		return
	}

	upload, download := summarizeSizes(payload.Logs)
	wifiName := normalizeWiFiName(payload.WiFiName)
	conn, err := s.insertConnection(payload.SystemIP, wifiName, upload, download)
	if err != nil {
		log.Printf("insert failed: %v", err)
		http.Error(w, "failed to persist payload", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, conn)
}

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleIngest(w, r)
	case http.MethodGet:
		s.handleListConnections(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func summarizeSizes(logs []TrafficLog) (upload uint64, download uint64) {
	for _, l := range logs {
		switch l.Direction {
		case DirectionUpload:
			upload += l.PacketSize
		case DirectionDownload:
			download += l.PacketSize
		}
	}
	return upload, download
}

func normalizeWiFiName(name string) string {
	v := strings.TrimSpace(name)
	if v == "" {
		return "unknown"
	}
	return v
}

func (s *Server) insertConnection(ip, wifiname string, uploadSize, downloadSize uint64) (Connection, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Connection{}, err
	}
	defer tx.Rollback()

	var prevTotalUpload uint64
	var prevTotalDownload uint64

	err = tx.QueryRow(`
SELECT total_upload, total_download
FROM connections
WHERE ip = ?
ORDER BY id DESC
LIMIT 1
`, ip).Scan(&prevTotalUpload, &prevTotalDownload)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Connection{}, err
	}
	if errors.Is(err, sql.ErrNoRows) {
		prevTotalUpload = 0
		prevTotalDownload = 0
	}

	newTotalUpload := prevTotalUpload + uploadSize
	newTotalDownload := prevTotalDownload + downloadSize

	now := time.Now().UTC()
	res, err := tx.Exec(`
INSERT INTO connections (ip, wifiname, download_size, upload_size, total_download, total_upload, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, ip, wifiname, downloadSize, uploadSize, newTotalDownload, newTotalUpload, now)
	if err != nil {
		return Connection{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Connection{}, err
	}

	if err := tx.Commit(); err != nil {
		return Connection{}, err
	}

	return Connection{
		ID:            id,
		IP:            ip,
		WiFiName:      wifiname,
		DownloadSize:  downloadSize,
		UploadSize:    uploadSize,
		TotalDownload: newTotalDownload,
		TotalUpload:   newTotalUpload,
		CreatedAt:     now,
	}, nil
}

func (s *Server) handleListConnections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query, limit := buildConnectionListQuery(r)
	rows, err := s.db.Query(query, limit)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items, err := scanConnections(rows)
	if err != nil {
		http.Error(w, "scan failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func buildConnectionListQuery(r *http.Request) (string, int) {
	q := r.URL.Query()
	orderBy := make([]string, 0, 3)

	if ord := normalizeOrder(q.Get("total_download")); ord != "" {
		orderBy = append(orderBy, "total_download "+ord)
	}
	if ord := normalizeOrder(q.Get("total_upload")); ord != "" {
		orderBy = append(orderBy, "total_upload "+ord)
	}
	if len(orderBy) == 0 {
		if sortBy := normalizeSortBy(q.Get("sort_by")); sortBy != "" {
			if ord := normalizeOrder(q.Get("order")); ord != "" {
				orderBy = append(orderBy, sortBy+" "+ord)
			}
		}
	}
	if len(orderBy) == 0 {
		orderBy = append(orderBy, "id DESC")
	} else {
		orderBy = append(orderBy, "id DESC")
	}

	limit := parseLimit(q.Get("limit"), 500)
	query := `
SELECT id, ip, wifiname, download_size, upload_size, total_download, total_upload, created_at
FROM connections
ORDER BY ` + strings.Join(orderBy, ", ") + `
LIMIT ?
`
	return query, limit
}

func normalizeSortBy(value string) string {
	switch strings.TrimSpace(value) {
	case "total_download":
		return "total_download"
	case "total_upload":
		return "total_upload"
	default:
		return ""
	}
}

func normalizeOrder(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ASC":
		return "ASC"
	case "DESC":
		return "DESC"
	default:
		return ""
	}
}

func parseLimit(value string, fallback int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	if n > 5000 {
		return 5000
	}
	return n
}

func scanConnections(rows *sql.Rows) ([]Connection, error) {
	items := make([]Connection, 0, 128)
	for rows.Next() {
		var c Connection
		if err := rows.Scan(
			&c.ID,
			&c.IP,
			&c.WiFiName,
			&c.DownloadSize,
			&c.UploadSize,
			&c.TotalDownload,
			&c.TotalUpload,
			&c.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Server) handleConnectionByIPSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ip := strings.TrimPrefix(r.URL.Path, "/api/v1/connections/")
	ip = strings.TrimSpace(ip)
	if ip == "" || strings.Contains(ip, "/") {
		http.Error(w, "invalid ip path", http.StatusBadRequest)
		return
	}

	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	history, err := s.listConnectionsByIP(ip, 5000)
	if err != nil {
		log.Printf("history query failed: %v", err)
		_ = ws.WriteJSON(map[string]any{"type": "error", "message": "history query failed"})
		return
	}
	lastID := int64(0)
	if len(history) > 0 {
		lastID = history[len(history)-1].ID
	}

	if err := ws.WriteJSON(map[string]any{
		"type": "history",
		"ip":   ip,
		"data": history,
	}); err != nil {
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		newRows, err := s.listConnectionsByIPAfterID(ip, lastID, 500)
		if err != nil {
			_ = ws.WriteJSON(map[string]any{"type": "error", "message": "stream query failed"})
			return
		}
		if len(newRows) == 0 {
			continue
		}
		lastID = newRows[len(newRows)-1].ID
		if err := ws.WriteJSON(map[string]any{
			"type": "update",
			"ip":   ip,
			"data": newRows,
		}); err != nil {
			return
		}
	}
}

func (s *Server) listConnectionsByIP(ip string, limit int) ([]Connection, error) {
	rows, err := s.db.Query(`
SELECT id, ip, wifiname, download_size, upload_size, total_download, total_upload, created_at
FROM connections
WHERE ip = ?
ORDER BY id ASC
LIMIT ?
`, ip, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConnections(rows)
}

func (s *Server) listConnectionsByIPAfterID(ip string, afterID int64, limit int) ([]Connection, error) {
	rows, err := s.db.Query(`
SELECT id, ip, wifiname, download_size, upload_size, total_download, total_upload, created_at
FROM connections
WHERE ip = ? AND id > ?
ORDER BY id ASC
LIMIT ?
`, ip, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConnections(rows)
}

func (s *Server) runMidnightReset() {
	for {
		wait := timeUntilNextMidnight(time.Now())
		timer := time.NewTimer(wait)
		<-timer.C
		timer.Stop()

		if err := s.resetConnections(); err != nil {
			log.Printf("midnight reset failed: %v", err)
			continue
		}
		log.Println("midnight reset completed: connections table cleared")
	}
}

func (s *Server) resetConnections() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM connections;`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sqlite_sequence WHERE name = 'connections';`); err != nil {
		return err
	}
	return tx.Commit()
}

func timeUntilNextMidnight(now time.Time) time.Duration {
	y, m, d := now.Date()
	location := now.Location()
	next := time.Date(y, m, d+1, 0, 0, 0, 0, location)
	return next.Sub(now)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response failed: %v", err)
	}
}

func waitForShutdown(httpServer *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func loadConfig() string {
	_ = godotenv.Load()
	raw := strings.TrimSpace(os.Getenv("SERVER_CLIENT"))
	if raw == "" {
		return defaultAddr
	}

	// Accept either ":5000"/"0.0.0.0:5000" or full URL.
	if strings.Contains(raw, "://") {
		if _, addr, err := net.SplitHostPort(strings.TrimPrefix(raw, "http://")); err == nil {
			return ":" + addr
		}
		parts := strings.Split(raw, "://")
		if len(parts) == 2 {
			hostPort := parts[1]
			if idx := strings.Index(hostPort, "/"); idx >= 0 {
				hostPort = hostPort[:idx]
			}
			if _, port, err := net.SplitHostPort(hostPort); err == nil {
				return ":" + port
			}
		}
	}

	return raw
}
