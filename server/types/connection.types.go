package types

type CreateConnectionBodyDTO struct {
	SystemIP    string       `json:"system_ip" form:"system_ip" validate:"required"`
	HostName    string       `json:"host_name,omitempty" form:"host_name"`
	WiFiName    string       `json:"wifi_name,omitempty" form:"wifi_name"`
	CollectedAt string       `json:"collected_at" form:"collected_at"`
	Logs        []TrafficLog `json:"logs" form:"logs"`
}

type ConnectionResponseDTO struct {
	ID            uint   `json:"id"`
	IP            string `json:"ip"`
	WiFiName      string `json:"wifi_name"`
	HostName      string `json:"host_name"`
	DownloadSize  uint64 `json:"download_size"`
	UploadSize    uint64 `json:"upload_size"`
	TotalDownload uint64 `json:"total_download"`
	TotalUpload   uint64 `json:"total_upload"`
	CollectedAt   string `json:"collected_at"`
}

type GetConnectionsQueryParamsDTO struct {
	PaginationQueryParams
	WiFiName      string `json:"wifi_name" form:"wifi_name"`
	HostName      string `json:"host_name" form:"host_name"`
	TotalUpload   string `json:"total_upload" form:"total_upload"`
	TotalDownload string `json:"total_download" form:"total_download"`
	AfterID       *uint  `json:"after_id" form:"after_id"`
}
