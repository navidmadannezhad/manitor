package types

import "time"

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
	SystemIP  string       `json:"system_ip" validate:"required"`
	HostName  string       `json:"host_name,omitempty"`
	WiFiName  string       `json:"wifi_name,omitempty"`
	Collected time.Time    `json:"collected_at"`
	Logs      []TrafficLog `json:"logs"`
}

type QueryParameters struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
