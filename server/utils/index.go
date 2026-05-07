package utils

import (
	"os"

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
