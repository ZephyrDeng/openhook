package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr       string
	DatabasePath   string
	RequestTimeout time.Duration
	AdminToken     string
	PublicBaseURL  string
}

func Load() Config {
	return Config{
		HTTPAddr:       env("OPENHOOK_ADDR", ":8080"),
		DatabasePath:   env("OPENHOOK_DB", "openhook.db"),
		RequestTimeout: envDuration("OPENHOOK_REQUEST_TIMEOUT", 10*time.Second),
		AdminToken:     os.Getenv("OPENHOOK_ADMIN_TOKEN"),
		PublicBaseURL:  os.Getenv("OPENHOOK_PUBLIC_BASE_URL"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
