package infra

import (
	"fmt"
	"log"
	"manitor-server/models"
	"manitor-server/utils"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB_HOST = utils.GetFromEnv("DB_HOST")
var DB_PORT = utils.GetFromEnv("DB_PORT")
var DB_USER = utils.GetFromEnv("DB_USER")
var DB_PASSWORD = utils.GetFromEnv("DB_PASSWORD")
var DB_NAME = utils.GetFromEnv("DB_NAME")
var DB_SSL_MODE = utils.GetFromEnv("DB_SSL_MODE")

var DB *gorm.DB

func InitiateDatabaseConnection() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		DB_HOST,
		DB_USER,
		DB_PASSWORD,
		DB_NAME,
		DB_PORT,
		DB_SSL_MODE,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	DB = db
	fmt.Println("Database - Connected!")
}

func InitiateMigration() {
	if DB == nil {
		InitiateDatabaseConnection()
	}

	err := DB.AutoMigrate(
		&models.Connection{},
	)
	if err != nil {
		log.Fatal("Migration - Failed ...")
	}
	fmt.Println("Migration - Done!")
}
