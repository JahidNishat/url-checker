package main

import (
	"os"
	"strconv"
)

type AppConfig struct {
	RedisAddr     string
	WorkerTimeout int
	HTTPTimeout   int
	MaxRetries    int
	ResultsToKeep int
}

func LoadConfig() AppConfig {
	return AppConfig{
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		WorkerTimeout: getEnvInt("WORKER_TIMEOUT", 1),
		HTTPTimeout:   getEnvInt("HTTP_TIMEOUT", 5),
		MaxRetries:    getEnvInt("MAX_RETRIES", 5),
		ResultsToKeep: getEnvInt("RESULTS_TO_KEEP", 10000),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
