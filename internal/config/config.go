package config

import (
	"log/slog"
	"os"
	"strconv"
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

	JWTSecret            string
	JWTExpiry            time.Duration
	TokenMagicLinkExpiry time.Duration

	MailerSMTPHost  string
	MailerSMTPPort  int
	MailerIMAPHost  string
	MailerIMAPPort  int
	MailerUsername  string
	MailerPassword  string
	MailerEmailFrom string

	SupportEmail string
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

		JWTSecret:            envRequired("JWT_SECRET"),
		JWTExpiry:            envDuration("JWT_EXPIRY", 168*time.Hour), // 7 days default
		TokenMagicLinkExpiry: envDuration("TOKEN_MAGIC_LINK_EXPIRY", 10*time.Minute),

		MailerSMTPHost:  envString("MAILER_SMTP_HOST", ""),
		MailerSMTPPort:  envInt("MAILER_SMTP_PORT", 587),
		MailerIMAPHost:  envString("MAILER_IMAP_HOST", ""),
		MailerIMAPPort:  envInt("MAILER_IMAP_PORT", 993),
		MailerUsername:  envString("MAILER_USERNAME", ""),
		MailerPassword:  envString("MAILER_PASSWORD", ""),
		MailerEmailFrom: envString("MAILER_EMAIL_FROM", ""),

		SupportEmail: envString("SUPPORT_EMAIL", ""),
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

		MailerEmailFrom: c.MailerEmailFrom,
		SupportEmail:    c.SupportEmail,
	}
}

func envString(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		value = def
	}
	return value
}

func envInt(key string, def int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return def
	}
	i, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		slog.Warn("config invalid integer, using default", "key", key, "value", value, "default", def)
		return def
	}
	return int(i)
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
