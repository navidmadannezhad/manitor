package utils

import (
	"manitor-server/types"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func init() {
	// اگر فایل .env وجود داشت لود کن
	if _, err := os.Stat(".env"); err == nil {
		godotenv.Load()
	}
}

func GetFromEnv(key string) string {
	return os.Getenv(key)
}

func ResolveError(err error) gin.H {
	return gin.H{
		"error": err.Error(),
	}
}

func ResolveResponse(data any) gin.H {
	return gin.H{
		"result": data,
	}
}

func GetTransferSizes(logs []types.TrafficLog) (uploadSize uint64, downloadSize uint64) {
	for _, log := range logs {
		switch log.Direction {
		case types.DirectionUpload:
			uploadSize = uploadSize + log.PacketSize
		case types.DirectionDownload:
			downloadSize = downloadSize + log.PacketSize
		}
	}

	return uploadSize, downloadSize
}

func GetUnknownIfEmpty(value string) string {
	var trimmedValue string = strings.TrimSpace(value)
	if trimmedValue == "" {
		return "unknown"
	}

	return trimmedValue
}

func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 10
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}
