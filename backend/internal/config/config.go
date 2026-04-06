package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	ServerPort    string
	JWTSecret     string
	LLMBaseURL    string
	LLMAPIKey     string
	LLMModel      string
	LLMTimeoutSec string
}

func Load() *Config {
	godotenv.Load()

	return &Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBName:        getEnv("DB_NAME", "social_app"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		LLMBaseURL:    getEnv("LLM_BASE_URL", "https://api.openai.com/v1"),
		LLMAPIKey:     getEnv("LLM_API_KEY", ""),
		LLMModel:      getEnv("LLM_MODEL", "gpt-4o-mini"),
		LLMTimeoutSec: getEnv("LLM_TIMEOUT_SECONDS", "20"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
