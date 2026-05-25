package config

import (
	"os"
	"time"
)

type Config struct {
	Crawl4AIAPIURL string
	ListenAddr     string
	RequestTimeout time.Duration
}

func Load() *Config {
	return &Config{
		Crawl4AIAPIURL: getEnv("CRAWL4AI_API_URL", "http://crawl4ai:11235"),
		ListenAddr:     getEnv("LISTEN_ADDR", ":8080"),
		RequestTimeout: parseDuration(getEnv("REQUEST_TIMEOUT", "150s")),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return 150 * time.Second
}
