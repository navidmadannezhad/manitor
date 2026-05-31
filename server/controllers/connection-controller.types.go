package controllers

import (
	"manitor-server/types"
)

type CreateConnectionBodyDTO struct {
	SystemIP    string             `json:"system_ip" form:"system_ip" validate:"required"`
	HostName    string             `json:"host_name,omitempty" form:"host_name"`
	WiFiName    string             `json:"wifi_name,omitempty" form:"wifi_name"`
	CollectedAt string             `json:"collected_at" form:"collected_at"`
	Logs        []types.TrafficLog `json:"logs" form:"logs"`
}

type GetConnectionsQueryParamsDTO struct {
	WiFiName      string `json:"wifi_name" form:"wifi_name"`
	HostName      string `json:"host_name" form:"host_name"`
	TotalUpload   string `json:"total_upload" form:"total_upload" binding:"oneof=asc desc"`
	TotalDownload string `json:"total_download" form:"total_download" binding:"oneof=asc desc"`
}
