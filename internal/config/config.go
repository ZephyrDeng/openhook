package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr           string
	DatabasePath       string
	RequestTimeout     time.Duration
	AdminToken         string
	PublicBaseURL      string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubAuthURL      string
	GitHubTokenURL     string
	GitHubUserURL      string
	SessionTTL         time.Duration
	RepositoryURL      string
}

func Load() Config {
	return Config{
		HTTPAddr:           env("OPENHOOK_ADDR", ":8080"),
		DatabasePath:       env("OPENHOOK_DB", "openhook.db"),
		RequestTimeout:     envDuration("OPENHOOK_REQUEST_TIMEOUT", 10*time.Second),
		AdminToken:         os.Getenv("OPENHOOK_ADMIN_TOKEN"),
		PublicBaseURL:      os.Getenv("OPENHOOK_PUBLIC_BASE_URL"),
		GitHubClientID:     os.Getenv("OPENHOOK_GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("OPENHOOK_GITHUB_CLIENT_SECRET"),
		GitHubAuthURL:      env("OPENHOOK_GITHUB_AUTH_URL", "https://github.com/login/oauth/authorize"),
		GitHubTokenURL:     env("OPENHOOK_GITHUB_TOKEN_URL", "https://github.com/login/oauth/access_token"),
		GitHubUserURL:      env("OPENHOOK_GITHUB_USER_URL", "https://api.github.com/user"),
		SessionTTL:         envDuration("OPENHOOK_SESSION_TTL", 30*24*time.Hour),
		RepositoryURL:      env("OPENHOOK_REPOSITORY_URL", "https://github.com/ZephyrDeng/openhook"),
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
