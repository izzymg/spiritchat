package config

import "os"

// ShouldRunIntegrations is a testing function, returns false if integrations shouldn't be run.
func ShouldRunIntegrations() bool {
	if env, exists := os.LookupEnv(
		"SPIRITTEST_INTEGRATIONS",
	); !exists || env == "FALSE" || env == "0" {
		return false
	}
	return true
}

/*
GetIntegrationsConfig is a testing function,
returns false if integrations shouldn't be run, or true, and integration config. */
func GetIntegrationsConfig() (*Config, bool) {
	if !ShouldRunIntegrations() {
		return nil, false
	}
	pgURL := os.Getenv("SPIRITTEST_PG_URL")
	redisURL := os.Getenv("SPIRITTEST_REDIS_URL")
	addr := os.Getenv("SPIRITTEST_ADDR")
	if len(pgURL) == 0 || len(redisURL) == 0 || len(addr) == 0 {
		panic("SPIRITTEST_PG_URL or SPIRITTEST_REDIS_URL or SPIRITTEST_ADDR empty")
	}

	return &Config{
		HTTPAddress: addr,
		PGURL:       pgURL,
		RedisURL:    redisURL,
	}, true
}

// Config stores configuration for the app.
type Config struct {
	HTTPAddress string
	CORSAllow   string
	PGURL       string
	RedisURL    string
}

// ParseEnv parses system environment variables, returning app configuration.
func ParseEnv() *Config {
	conf := &Config{
		HTTPAddress: "0.0.0.0:3000",
		CORSAllow:   "https://example.com",
		PGURL:       os.Getenv("SPIRITCHAT_PG_URL"),
		RedisURL:    os.Getenv("SPIRITCHAT_REDIS_URL"),
	}
	if addr, ok := os.LookupEnv("SPIRITCHAT_ADDRESS"); ok {
		conf.HTTPAddress = addr
	}

	if allow, ok := os.LookupEnv("SPIRITCHAT_CORS_ALLOW"); ok {
		conf.CORSAllow = allow
	}
	return conf
}