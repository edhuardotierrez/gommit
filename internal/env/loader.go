package env

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadFile load .env file if present and set variables in the environment.
func LoadFile() {
	envFile := ".env"
	workingDir, err := os.Getwd()
	if err == nil {
		envFile = filepath.Join(workingDir, envFile)
	}
	if envMap, errRead := godotenv.Read(envFile); errRead == nil {
		for key, value := range envMap {
			_ = os.Setenv(key, value)
		}
	}
}

func GetString(key string) string {
	return os.Getenv(key)
}
