package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultPort = 5000

var ErrEmptyAuthToken = errors.New("AUTH_TOKEN must be non-empty")

type Config struct {
	Port      int
	AuthToken string
	GroupJID  string
	AuthDBDSN string
	LogsDBDSN string
}

func LoadFromEnv() (Config, error) {
	if err := loadDotEnvCandidates([]string{".env", "../.env"}); err != nil {
		return Config{}, err
	}

	port := defaultPort

	rawPort := strings.TrimSpace(os.Getenv("PORT"))
	if rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil || parsed <= 0 || parsed > 65535 {
			return Config{}, fmt.Errorf("invalid PORT: %q (must be 1-65535)", rawPort)
		}
		port = parsed
	}

	authToken := strings.TrimSpace(os.Getenv("AUTH_TOKEN"))
	if authToken == "" {
		return Config{}, ErrEmptyAuthToken
	}

	return Config{
		Port:      port,
		AuthToken: authToken,
		GroupJID:  strings.TrimSpace(os.Getenv("GROUP_JID")),
		AuthDBDSN: envWithDefault("AUTH_DB_DSN", "file:auth.db?_foreign_keys=on"),
		LogsDBDSN: envWithDefault("LOGS_DB_DSN", "file:logs.db?_foreign_keys=on"),
	}, nil
}

func envWithDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
