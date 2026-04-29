package utils

import (
	"os"

	"github.com/joho/godotenv"
)

func GetFromEnv(key string) string {
	err := godotenv.Load()
	if err != nil {
		return ""
	}

	value := os.Getenv(key)
	return value
}
