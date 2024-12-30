package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type ARIConfig struct {
	Application  string
	Username     string
	Password     string
	URL          string
	WebsocketURL string
}

func GetAriConfig() ARIConfig {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return ARIConfig{
		Application:  os.Getenv("ARI_APP"),
		Username:     os.Getenv("ARI_USERNAME"),
		Password:     os.Getenv("ARI_PASSWORD"),
		URL:          os.Getenv("ARI_URL"),
		WebsocketURL: os.Getenv("ARI_WS_URL"),
	}
}
