package utils

import (
	"os"

	"github.com/joho/godotenv"
)

func init() {
	if _, err := os.Stat(".env"); err == nil {
		godotenv.Load()
	}
}

func GetFromEnv(key string) string {
	return os.Getenv(key)
}
