package config

import (
	"log"

	"github.com/joho/godotenv"
)

func loadEnvFile(path string) error {
	return godotenv.Load(path)
}

func LoadDotEnv() {
	for _, path := range []string{".env", "../../.env"} {
		if err := loadEnvFile(path); err == nil {
			log.Printf("loaded env from %s", path)
			return
		}
	}
	log.Print("no .env file found in expected locations")
}
