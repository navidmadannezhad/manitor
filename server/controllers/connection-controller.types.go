package controllers

import (
	"manitor-server/types"
	"time"
)

type CreateControllerBodyDTO struct {
	SystemIP    string             `json:"system_ip" validate:"required"`
	HostName    string             `json:"host_name,omitempty"`
	WiFiName    string             `json:"wifi_name,omitempty"`
	CollectedAt time.Time          `json:"collected_at"`
	Logs        []types.TrafficLog `json:"logs"`
}
