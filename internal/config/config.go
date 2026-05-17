package config

import "os"

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// REST API
	DatabaseURL   string
	RedisURL      string
	RestAPIPort   string
	LogFormat     string
	AllowedOrigin string

	// gRPC Server
	GRPCPort   string
	RestAPIURL string

	// gRPC Client
	ServerAddr string
}

// Load reads environment variables and returns a Config with defaults applied.
func Load() Config {
	return Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://inventory:secret@localhost:5432/inventory?sslmode=disable"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
		RestAPIPort:   getEnv("REST_API_PORT", "8080"),
		LogFormat:     getEnv("LOG_FORMAT", "text"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"),
		GRPCPort:      getEnv("GRPC_PORT", "50051"),
		RestAPIURL:    getEnv("REST_API_URL", "http://localhost:8080"),
		ServerAddr:    getEnv("SERVER_ADDR", "localhost:50051"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
