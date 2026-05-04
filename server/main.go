package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"manitor-server/utils"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

const (
	defaultAddr = ":5000"
)

var DB_HOST = utils.GetFromEnv("DB_HOST")
var DB_PORT = utils.GetFromEnv("DB_PORT")
var DB_USER = utils.GetFromEnv("DB_USER")
var DB_PASSWORD = utils.GetFromEnv("DB_PASSWORD")
var DB_NAME = utils.GetFromEnv("DB_NAME")
var DB_SSL_MODE = utils.GetFromEnv("DB_SSL_MODE")

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
	HostName  string       `json:"host_name,omitempty"`
	WiFiName  string       `json:"wifi_name,omitempty"`
	Collected time.Time    `json:"collected_at"`
	Logs      []TrafficLog `json:"logs"`
}

type Connection struct {
	ID            int64     `json:"id"`
	IP            string    `json:"ip"`
	WiFiName      string    `json:"wifi_name"`
	HostName      string    `json:"host_name"`
	DownloadSize  uint64    `json:"download_size"`
	UploadSize    uint64    `json:"upload_size"`
	TotalDownload uint64    `json:"total_download"`
	TotalUpload   uint64    `json:"total_upload"`
	CreatedAt     time.Time `json:"created_at"`
}

type Server struct {
	db      *sql.DB
	writeMu sync.Mutex
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	listenAddr := loadConfig()

	db, err := openDB()
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := ensureSchema(db); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}
	if err := ensureHostnameColumn(db); err != nil {
		log.Fatalf("migrate schema: %v", err)
	}

	s := &Server{db: db}
	go s.runMidnightReset()

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIngest)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/v1/connections/stream", s.handleSessionStreamSocket)
	mux.HandleFunc("/api/v1/connections", s.handleConnections)

	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: withCORS(mux),
	}

	go func() {
		log.Printf("server listening on %s", listenAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen failed: %v", err)
		}
	}()

	waitForShutdown(httpServer)
}

func openDB() (*sql.DB, error) {
	dsn := "host=" + DB_HOST +
		" port=" + DB_PORT +
		" user=" + DB_USER +
		" password=" + DB_PASSWORD +
		" dbname=" + DB_NAME +
		" sslmode=" + DB_SSL_MODE

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func ensureSchema(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS connections (
	id BIGSERIAL PRIMARY KEY,
	ip TEXT NOT NULL,
	wifiname TEXT NOT NULL,
	hostname TEXT NOT NULL DEFAULT '',
	download_size BIGINT NOT NULL,
	upload_size BIGINT NOT NULL,
	total_download BIGINT NOT NULL,
	total_upload BIGINT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_connections_ip ON connections(ip);
CREATE INDEX IF NOT EXISTS idx_connections_created_at ON connections(created_at);
CREATE INDEX IF NOT EXISTS idx_connections_ip_id ON connections(ip, id);
CREATE INDEX IF NOT EXISTS idx_connections_host_wifi_id ON connections(hostname, wifiname, id);
`)
	return err
}

func ensureHostnameColumn(db *sql.DB) error {
	var n int
	if err := db.QueryRow(`
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = current_schema()
	AND table_name = 'connections'
	AND column_name = 'hostname'
`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := db.Exec(`ALTER TABLE connections ADD COLUMN hostname TEXT NOT NULL DEFAULT ''`)
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
	hostName := normalizeHostName(payload.HostName)
	conn, err := s.insertConnection(payload.SystemIP, wifiName, hostName, upload, download)
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

func normalizeHostName(name string) string {
	v := strings.TrimSpace(name)
	if v == "" {
		return "unknown"
	}
	return v
}

func (s *Server) insertConnection(ip, wifiname, hostname string, uploadSize, downloadSize uint64) (Connection, error) {

	fmt.Println("Test log --")
	fmt.Println(DB_HOST)
	fmt.Println(DB_PORT)
	fmt.Println(DB_NAME)

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

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
WHERE hostname = $1 AND wifiname = $2
ORDER BY id DESC
LIMIT 1
`, hostname, wifiname).Scan(&prevTotalUpload, &prevTotalDownload)
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
	var id int64
	err = tx.QueryRow(`
INSERT INTO connections (ip, wifiname, hostname, download_size, upload_size, total_download, total_upload, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id
`, ip, wifiname, hostname, downloadSize, uploadSize, newTotalDownload, newTotalUpload, now).Scan(&id)
	if err != nil {
		return Connection{}, err
	}

	if err := tx.Commit(); err != nil {
		return Connection{}, err
	}

	fmt.Println("Insert Data")
	fmt.Println(hostname)

	return Connection{
		ID:            id,
		IP:            ip,
		WiFiName:      wifiname,
		HostName:      hostname,
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
	for i := range orderBy {
		orderBy[i] = prefixOrderExprLatestPerSession(orderBy[i])
	}

	limit := parseLimit(q.Get("limit"), 500)
	// One row per (display name, Wi‑Fi): latest row per pair. Cumulative totals live on that row.
	query := `
SELECT c.id, c.ip, c.wifiname, c.hostname, c.download_size, c.upload_size, c.total_download, c.total_upload, c.created_at
FROM connections c
INNER JOIN (
	SELECT hostname, wifiname, MAX(id) AS max_id
	FROM connections
	GROUP BY hostname, wifiname
) latest ON c.hostname = latest.hostname AND c.wifiname = latest.wifiname AND c.id = latest.max_id
ORDER BY ` + strings.Join(orderBy, ", ") + `
LIMIT $1
`
	return query, limit
}

// prefixOrderExprLatestPerSession maps list sort tokens to the aliased subquery in handleListConnections.
func prefixOrderExprLatestPerSession(expr string) string {
	expr = strings.TrimSpace(expr)
	for _, col := range []string{"total_download", "total_upload", "id", "created_at", "ip", "wifiname", "hostname", "download_size", "upload_size"} {
		prefix := col + " "
		if len(expr) > len(prefix) && strings.EqualFold(expr[:len(prefix)], prefix) {
			rest := strings.TrimSpace(expr[len(prefix):])
			if rest == "ASC" || rest == "DESC" {
				return "c." + col + " " + rest
			}
		}
	}
	return "c." + expr
}

func normalizeSortBy(value string) string {
	switch strings.TrimSpace(value) {
	case "total_download":
		return "total_download"
	case "total_upload":
		return "total_upload"
	case "hostname":
		return "hostname"
	case "wifiname":
		return "wifiname"
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
			&c.HostName,
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

func (s *Server) handleSessionStreamSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	if _, ok := q["host_name"]; !ok {
		http.Error(w, "host_name query parameter is required", http.StatusBadRequest)
		return
	}
	if _, ok := q["wifi_name"]; !ok {
		http.Error(w, "wifi_name query parameter is required", http.StatusBadRequest)
		return
	}
	host := normalizeHostName(q.Get("host_name"))
	wifi := normalizeWiFiName(q.Get("wifi_name"))

	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	history, err := s.listConnectionsBySession(host, wifi, 5000)
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
		"type":      "history",
		"host_name": host,
		"wifi_name": wifi,
		"data":      history,
	}); err != nil {
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		newRows, err := s.listConnectionsBySessionAfterID(host, wifi, lastID, 500)
		if err != nil {
			_ = ws.WriteJSON(map[string]any{"type": "error", "message": "stream query failed"})
			return
		}
		if len(newRows) == 0 {
			continue
		}
		lastID = newRows[len(newRows)-1].ID
		if err := ws.WriteJSON(map[string]any{
			"type":      "update",
			"host_name": host,
			"wifi_name": wifi,
			"data":      newRows,
		}); err != nil {
			return
		}
	}
}

func (s *Server) listConnectionsBySession(hostname, wifiname string, limit int) ([]Connection, error) {
	rows, err := s.db.Query(`
SELECT id, ip, wifiname, hostname, download_size, upload_size, total_download, total_upload, created_at
FROM connections
WHERE hostname = $1 AND wifiname = $2
ORDER BY id ASC
LIMIT $3
`, hostname, wifiname, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConnections(rows)
}

func (s *Server) listConnectionsBySessionAfterID(hostname, wifiname string, afterID int64, limit int) ([]Connection, error) {
	rows, err := s.db.Query(`
SELECT id, ip, wifiname, hostname, download_size, upload_size, total_download, total_upload, created_at
FROM connections
WHERE hostname = $1 AND wifiname = $2 AND id > $3
ORDER BY id ASC
LIMIT $4
`, hostname, wifiname, afterID, limit)
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
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`TRUNCATE TABLE connections RESTART IDENTITY;`); err != nil {
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

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
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
	raw := strings.TrimSpace(utils.GetFromEnv(("SERVER_CLIENT")))
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
