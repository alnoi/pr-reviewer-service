package config

import (
	"fmt"
	"net"
	"os"
)

type Config struct {
	HTTPPort           string
	DB                 DB
	MetricsPort        string
	PyroscopeEnabled   bool
	PyroscopeAddress   string
	JaegerCollectorURL string
}

type DB struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func Load() *Config {
	return &Config{
		HTTPPort:           getEnv("HTTP_PORT", "8080"),
		DB:                 loadDB(),
		MetricsPort:        getEnv("METRICS_PORT", "9100"),
		PyroscopeEnabled:   getEnv("PYROSCOPE_ENABLED", "false") == "true",
		PyroscopeAddress:   getEnv("PYROSCOPE_SERVER_ADDRESS", "http://pyroscope:4040"),
		JaegerCollectorURL: getEnv("JAEGER_COLLECTOR_URL", ""),
	}
}

func loadDB() DB {
	return DB{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		Name:     getEnv("DB_NAME", "pr_review"),
	}
}

func (d DB) DSN() string {
	hostPort := net.JoinHostPort(d.Host, d.Port)
	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		d.User, d.Password, hostPort, d.Name,
	)
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
