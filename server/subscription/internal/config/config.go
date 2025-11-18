package config

import (
	"fmt"
	"os"
	"strings"
)

// Config aggregates every tunable part of the application.
type Config struct {
	App     AppConfig
	DB      DBConfig
	Log     LogConfig
	Swagger SwaggerConfig
}

// AppConfig contains settings related to the HTTP server.
type AppConfig struct {
	Port string
	Env  string
}

// DBConfig represents PostgreSQL connection settings.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN builds the postgres connection string from the individual fields.
func (db DBConfig) DSN() string {
	host := db.Host
	if host == "" {
		host = "localhost"
	}

	port := db.Port
	if port == "" {
		port = "5432"
	}

	sslMode := db.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		db.User,
		db.Password,
		host,
		port,
		db.Name,
		sslMode,
	)
}

// LogConfig controls logger behavior.
type LogConfig struct {
	Level string
}

// SwaggerConfig configures the generated documentation.
type SwaggerConfig struct {
	Host string
}

// Load reads environment variables and validates the final configuration.
func Load() (Config, error) {
	cfg := Config{
		App: AppConfig{
			Port: getEnv("APP_PORT", "8080"),
			Env:  getEnv("APP_ENV", "dev"),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", ""),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", ""),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Log: LogConfig{
			Level: strings.ToLower(getEnv("LOG_LEVEL", "info")),
		},
		Swagger: SwaggerConfig{
			Host: getEnv("SWAGGER_HOST", ""),
		},
	}

	if cfg.Swagger.Host == "" {
		cfg.Swagger.Host = fmt.Sprintf("localhost:%s", cfg.App.Port)
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) validate() error {
	var missing []string

	if cfg.DB.User == "" {
		missing = append(missing, "DB_USER")
	}
	if cfg.DB.Password == "" {
		missing = append(missing, "DB_PASSWORD")
	}
	if cfg.DB.Name == "" {
		missing = append(missing, "DB_NAME")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	return nil
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	return value
}
