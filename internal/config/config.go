package config

import (
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName    string
	AppTagline string
	AppEnv     string
	AppURL     string
	Host       string
	Port       string

	DBDriver     string
	DBConnection string

	JWTSecret string
	JWTExpiry time.Duration

	MailerSMTPHost     string
	MailerSMTPPort     string
	MailerUsername     string
	MailerPassword     string
	MailerEmailFrom    string
	MailerEnvelopeFrom string
}

func Load() *Config {

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using environment variables")
	}

	cfg := &Config{
		AppName:    envString("APP_NAME", "Budgit"),
		AppTagline: envString("APP_TAGLINE", "Finance tracking made easy."),
		AppEnv:     envRequired("APP_ENV"),
		AppURL:     envRequired("APP_URL"),
		Host:       envString("HOST", "127.0.0.1"),
		Port:       envString("PORT", "9000"),

		DBDriver:     envString("DB_DRIVER", "sqlite"),
		DBConnection: envString("DB_CONNECTION", "./data/local.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"),

		JWTSecret: envRequired("JWT_SECRET"),
		JWTExpiry: envDuration("JWT_EXPIRY", 168*time.Hour), // 7 days default

		MailerSMTPHost:     envString("MAILER_SMTP_HOST", ""),
		MailerSMTPPort:     envString("MAILER_SMTP_PORT", ""),
		MailerUsername:     envString("MAILER_USERNAME", ""),
		MailerPassword:     envString("MAILER_PASSWORD", ""),
		MailerEmailFrom:    envString("MAILER_EMAIL_FROM", ""),
		MailerEnvelopeFrom: envString("MAILER_ENVELOPE_FROM", ""),
	}

	return cfg
}

func (cfg *Config) IsProduction() bool {
	return cfg.AppEnv == "production"
}

// Sanitized returns a copy of the config with only public/safe fields.
// All secrets, credentials, and sensitive data are excluded.
// Safe to expose in ctx, templates and client-facing contexts.
func (c *Config) Sanitized() *Config {
	return &Config{
		AppName:    c.AppName,
		AppEnv:     c.AppEnv,
		AppURL:     c.AppURL,
		Port:       c.Port,
		AppTagline: c.AppTagline,

		MailerEmailFrom:    c.MailerEmailFrom,
		MailerEnvelopeFrom: c.MailerEnvelopeFrom,
	}
}

func envString(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		value = def
	}
	return value
}

func envDuration(key string, def time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return def
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		slog.Warn("config invalid duration, using default", "key", key, "value", value, "default", def)
		return def
	}
	return duration
}

func envRequired(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	slog.Error("config required env var missing", "key", key)
	os.Exit(1)
	return ""
}
