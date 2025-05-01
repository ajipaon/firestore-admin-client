package config

import (
	"os"
)

type Config struct {
	Port             string `json:"port"`
	FirebaseCredFile string `json:"firebase_cred_file"`
	FirebaseApiKey   string `json:"firebase_api_key"`
	AdminUsername    string `json:"admin_username"`
	AdminPassword    string `json:"admin_password"`
}

func LoadConfig() Config {
	return Config{
		Port:             GetEnv("PORT", "8080"),
		FirebaseCredFile: GetEnv("FIREBASE_CRED_FILE", "firebase-credential.json"),
		FirebaseApiKey:   GetEnv("FIREBASE_API_KEY", ""),
		AdminUsername:    GetEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:    GetEnv("ADMIN_PASSWORD", "admin"),
	}
}

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
