package config

import (
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	Secrets map[string]string
}

var (
	once     sync.Once
	instance *Config
)

func GetConfig() *Config {
	once.Do(func() {
		instance = &Config{
			Secrets: make(map[string]string),
		}

		// Load .env file and populate Secrets
		loadSecrets(instance)
	})
	return instance
}

/*func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}*/

func loadSecrets(cfg *Config) {
	// Load .env file
	envMap, err := godotenv.Read()
	if err != nil {
		// Handle error (e.g., log it)
		return
	}

	// Populate Secrets with all entries from .env
	for key, value := range envMap {
		cfg.Secrets[key] = value
	}
}

func (c *Config) GetSecret(key string) string {
	return c.Secrets[key]
}

func SetSecret(key, value string) {
	instance.Secrets[key] = value
}
