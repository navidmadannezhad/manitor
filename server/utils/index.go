package utils

import (
	"manitor-server/types"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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
		"error": err,
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
