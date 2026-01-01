package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv      string
	AppPort     string
	AppName     string
	
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	
	JWTSecret           string
	JWTExpiration       time.Duration
	JWTRefreshExpiration time.Duration
	
	CorsOrigins string
	LogLevel    string
	
	// Redis Cache Configuration
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	CacheEnabled  bool
}

func Load() *Config {
	return &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		AppPort:     getEnv("APP_PORT", "8080"),
		AppName:     getEnv("APP_NAME", "tech-erp-api"),
		
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "erp"),
		DBPassword: getEnv("DB_PASSWORD", "erp123"),
		DBName:     getEnv("DB_NAME", "tech_erp"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		
		JWTSecret:           getEnv("JWT_SECRET", "your-super-secret-key"),
		JWTExpiration:       parseDuration(getEnv("JWT_EXPIRATION", "8h")),
		JWTRefreshExpiration: parseDuration(getEnv("JWT_REFRESH_EXPIRATION", "168h")),
		
		CorsOrigins: getEnv("CORS_ORIGINS", "*"),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),
		
		// Redis Cache Configuration
		RedisHost:     getEnv("REDIS_HOST", "redis-service"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       parseInt(getEnv("REDIS_DB", "0")),
		CacheEnabled:  parseBool(getEnv("CACHE_ENABLED", "true")),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 8 * time.Hour
	}
	return d
}

func parseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func parseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return b
}

func (c *Config) GetDSN() string {
	return "host=" + c.DBHost +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" port=" + c.DBPort +
		" sslmode=" + c.DBSSLMode +
		" TimeZone=America/Sao_Paulo"
}
