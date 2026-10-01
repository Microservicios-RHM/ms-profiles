package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Settings struct {
	Port     int
	LogLevel string

	DBHost                string
	DBPort                int
	DBName                string
	DBUser                string
	DBPassword            string
	DBPoolMax             int32
	DBConnectMaxAttempts  int
	DBConnectRetryDelayMs int

	BrokerURL                 string
	BrokerExchange            string
	BrokerQueue               string
	BrokerConnectMaxAttempts  int
	BrokerConnectRetryDelayMs int
}

func (s Settings) DatabaseDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", s.DBUser, s.DBPassword, s.DBHost, s.DBPort, s.DBName)
}

func Load() (Settings, error) {
	if _, err := os.Stat(".env"); err == nil {
		_ = godotenv.Load()
	}

	required := []string{"DB_HOST", "DB_NAME", "DB_USER", "DB_PASSWORD", "BROKER_URL"}
	var missing []string
	for _, key := range required {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return Settings{}, fmt.Errorf("configuración inválida o incompleta: %s", strings.Join(missing, ", "))
	}

	brokerURL := os.Getenv("BROKER_URL")
	if !strings.HasPrefix(brokerURL, "amqp://") && !strings.HasPrefix(brokerURL, "amqps://") {
		return Settings{}, fmt.Errorf("BROKER_URL debe ser una URL amqp:// o amqps://")
	}

	return Settings{
		Port:     envInt("PORT", 8080),
		LogLevel: envString("LOG_LEVEL", "info"),

		DBHost:                os.Getenv("DB_HOST"),
		DBPort:                envInt("DB_PORT", 5432),
		DBName:                os.Getenv("DB_NAME"),
		DBUser:                os.Getenv("DB_USER"),
		DBPassword:            os.Getenv("DB_PASSWORD"),
		DBPoolMax:             int32(envInt("DB_POOL_MAX", 5)),
		DBConnectMaxAttempts:  envInt("DB_CONNECT_MAX_ATTEMPTS", 5),
		DBConnectRetryDelayMs: envInt("DB_CONNECT_RETRY_DELAY_MS", 1000),

		BrokerURL:                 brokerURL,
		BrokerExchange:            envString("BROKER_EXCHANGE", "rhm.events"),
		BrokerQueue:               envString("BROKER_QUEUE", "perfiles.queue"),
		BrokerConnectMaxAttempts:  envInt("BROKER_CONNECT_MAX_ATTEMPTS", 5),
		BrokerConnectRetryDelayMs: envInt("BROKER_CONNECT_RETRY_DELAY_MS", 1000),
	}, nil
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
