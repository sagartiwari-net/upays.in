package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	AppEnv   string
	AppPort  string
	AppURL   string
	DBHost   string
	DBPort   string
	DBName   string
	DBUser   string
	DBPass   string
	RedisURL string
	LogLevel string

	PhonePeMerchantID string
	PhonePeSaltKey    string
	PhonePeSaltIndex  string
	PhonePeEnv        string

	OrderExpiryMinutes     int
	SignatureMaxAgeMinutes int

	PaymentProvider string
	UPIID           string
	UPIPayeeName    string

	IMAPHost            string
	IMAPPort            int
	IMAPUser            string
	IMAPPassword        string
	IMAPSenderFilter    string
	EmailPollIntervalSec int
	ProfileBootstrap     bool

	JWTSecret string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig()

	viper.AutomaticEnv()

	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_PORT", "8090")
	viper.SetDefault("APP_URL", "http://localhost:8090")
	viper.SetDefault("DB_HOST", "127.0.0.1")
	viper.SetDefault("DB_PORT", "3306")
	viper.SetDefault("DB_NAME", "upipays")
	viper.SetDefault("DB_USER", "upipays_user")
	viper.SetDefault("DB_PASSWORD", "")
	viper.SetDefault("REDIS_URL", "redis://127.0.0.1:6379")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("PHONEPE_MERCHANT_ID", "")
	viper.SetDefault("PHONEPE_SALT_KEY", "")
	viper.SetDefault("PHONEPE_SALT_INDEX", "1")
	viper.SetDefault("PHONEPE_ENV", "PRODUCTION")
	viper.SetDefault("ORDER_EXPIRY_MINUTES", 5)
	viper.SetDefault("SIGNATURE_MAX_AGE_MINUTES", 5)
	viper.SetDefault("PAYMENT_PROVIDER", "upi_email")
	viper.SetDefault("UPI_ID", "")
	viper.SetDefault("UPI_PAYEE_NAME", "UPIPays")
	viper.SetDefault("IMAP_HOST", "imap.gmail.com")
	viper.SetDefault("IMAP_PORT", 993)
	viper.SetDefault("IMAP_USER", "")
	viper.SetDefault("IMAP_PASSWORD", "")
	viper.SetDefault("IMAP_SENDER_FILTER", "hdfcbank")
	viper.SetDefault("EMAIL_POLL_INTERVAL_SEC", 30)
	viper.SetDefault("PROFILE_BOOTSTRAP", true)
	viper.SetDefault("JWT_SECRET", "change_me_in_development")

	cfg := &Config{
		AppEnv:   viper.GetString("APP_ENV"),
		AppPort:  viper.GetString("APP_PORT"),
		AppURL:   strings.TrimRight(viper.GetString("APP_URL"), "/"),
		DBHost:   viper.GetString("DB_HOST"),
		DBPort:   viper.GetString("DB_PORT"),
		DBName:   viper.GetString("DB_NAME"),
		DBUser:   viper.GetString("DB_USER"),
		DBPass:   viper.GetString("DB_PASSWORD"),
		RedisURL: viper.GetString("REDIS_URL"),
		LogLevel: viper.GetString("LOG_LEVEL"),

		PhonePeMerchantID: viper.GetString("PHONEPE_MERCHANT_ID"),
		PhonePeSaltKey:    viper.GetString("PHONEPE_SALT_KEY"),
		PhonePeSaltIndex:  viper.GetString("PHONEPE_SALT_INDEX"),
		PhonePeEnv:        viper.GetString("PHONEPE_ENV"),

		OrderExpiryMinutes:     viper.GetInt("ORDER_EXPIRY_MINUTES"),
		SignatureMaxAgeMinutes: viper.GetInt("SIGNATURE_MAX_AGE_MINUTES"),

		PaymentProvider:      viper.GetString("PAYMENT_PROVIDER"),
		UPIID:                viper.GetString("UPI_ID"),
		UPIPayeeName:         viper.GetString("UPI_PAYEE_NAME"),
		IMAPHost:             viper.GetString("IMAP_HOST"),
		IMAPPort:             viper.GetInt("IMAP_PORT"),
		IMAPUser:             viper.GetString("IMAP_USER"),
		IMAPPassword:         viper.GetString("IMAP_PASSWORD"),
		IMAPSenderFilter:     viper.GetString("IMAP_SENDER_FILTER"),
		EmailPollIntervalSec: viper.GetInt("EMAIL_POLL_INTERVAL_SEC"),
		ProfileBootstrap:     viper.GetBool("PROFILE_BOOTSTRAP"),
		JWTSecret:            viper.GetString("JWT_SECRET"),
	}

	if cfg.DBPass == "" && cfg.AppEnv == "production" {
		return nil, fmt.Errorf("DB_PASSWORD is required in production")
	}
	if cfg.JWTSecret == "change_me_in_development" && cfg.AppEnv == "production" {
		return nil, fmt.Errorf("JWT_SECRET must be set in production")
	}

	return cfg, nil
}

func (c *Config) EmailWorkerEnabled() bool {
	return c.PaymentProvider == "upi_email"
}

func (c *Config) EncryptSecret() string {
	return c.JWTSecret
}

func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC&multiStatements=true",
		c.DBUser,
		c.DBPass,
		c.DBHost,
		c.DBPort,
		c.DBName,
	)
}
