package models

type Connection struct {
	BaseModel
	IP            string `json:"ip" gorm:"ip"`
	WiFiName      string `json:"wifi_name" gorm:"wifi_name"`
	HostName      string `json:"host_name" gorm:"host_name"`
	DownloadSize  uint64 `json:"download_size" gorm:"download_size"`
	UploadSize    uint64 `json:"upload_size" gorm:"upload_size"`
	TotalDownload uint64 `json:"total_download" gorm:"total_download"`
	TotalUpload   uint64 `json:"total_upload" gorm:"total_upload"`
}
