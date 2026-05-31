package controllers

import (
	"manitor-server/types"
)

type CreateControllerBodyDTO struct {
	SystemIP    string             `json:"system_ip" form:"system_ip" validate:"required"`
	HostName    string             `json:"host_name,omitempty" form:"host_name"`
	WiFiName    string             `json:"wifi_name,omitempty" form:"wifi_name"`
	CollectedAt string             `json:"collected_at" form:"collected_at"`
	Logs        []types.TrafficLog `json:"logs" form:"logs"`
}
