package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	Port string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file. Using environment variables.")
	}

	cfg := &Config{
		Port: os.Getenv("PORT"),
	}

	return cfg, nil
}
