package models

import "time"

type Connection struct {
	BaseModel
	IP            string    `gorm:"ip"`
	WifiName      string    `gorm:"wifi_name"`
	HostName      string    `gorm:"host_name"`
	DownloadSize  uint64    `gorm:"download_size"`
	UploadSize    uint64    `gorm:"upload_size"`
	TotalDownload uint64    `gorm:"total_download"`
	TotalUpload   uint64    `gorm:"total_upload"`
	CollectedAt   time.Time `gorm:"collected_at"`
}
