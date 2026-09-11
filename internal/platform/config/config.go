package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	JWT          JWTConfig
	QR           QRConfig
	Notification NotificationConfig
	Google       GoogleConfig
}

type GoogleConfig struct {
	ClientID string `env:"GOOGLE_CLIENT_ID"`
}

type NotificationConfig struct {
	FCMAPIKey string `env:"FCM_API_KEY"`
}

type ServerConfig struct {
	Host            string        `env:"SERVER_HOST" envDefault:"0.0.0.0"`
	Port            int           `env:"SERVER_PORT" envDefault:"8080"`
	ReadTimeout     time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout    time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"30s"`
	ShutdownTimeout time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type DatabaseConfig struct {
	Host         string `env:"DB_HOST" envDefault:"localhost"`
	Port         int    `env:"DB_PORT" envDefault:"5432"`
	User         string `env:"DB_USER" envDefault:"turnero"`
	Password     string `env:"DB_PASSWORD" envDefault:"turnero_dev"`
	Name         string `env:"DB_NAME" envDefault:"turnero"`
	SSLMode      string `env:"DB_SSL_MODE" envDefault:"disable"`
	MaxOpenConns int    `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	MaxIdleConns int    `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type JWTConfig struct {
	Secret             string        `env:"JWT_SECRET,required"`
	AccessTokenExpiry  time.Duration `env:"JWT_ACCESS_EXPIRY" envDefault:"15m"`
	RefreshTokenExpiry time.Duration `env:"JWT_REFRESH_EXPIRY" envDefault:"168h"`
	Issuer             string        `env:"JWT_ISSUER" envDefault:"turnero"`
}

type QRConfig struct {
	BaseURL   string `env:"QR_BASE_URL" envDefault:"https://turnero.app"`
	OutputDir string `env:"QR_OUTPUT_DIR" envDefault:"/var/data/qr"`
}

func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required and must not be empty")
	}
	return nil
}
