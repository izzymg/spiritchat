package config

import (
	"os"
	"strconv"
)

func getPostCooldownEnv(env string) int {
	env, found := os.LookupEnv(env)
	if !found {
		return 0
	}
	cooldown, err := strconv.ParseInt(env, 10, 64)
	if err != nil {
		return 0
	}
	return int(cooldown)
}

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
returns false if integrations shouldn't be run, or true, and integration config.
*/
func GetIntegrationsConfig() (*SpiritConfig, bool) {
	if !ShouldRunIntegrations() {
		return nil, false
	}
	pgURL := os.Getenv("SPIRITTEST_PG_URL")
	redisURL := os.Getenv("SPIRITTEST_REDIS_URL")
	addr := os.Getenv("SPIRITTEST_ADDR")
	if len(pgURL) == 0 || len(redisURL) == 0 || len(addr) == 0 {
		panic("SPIRITTEST_PG_URL or SPIRITTEST_REDIS_URL or SPIRITTEST_ADDR empty")
	}

	return &SpiritConfig{
		HTTPAddress:         addr,
		PGURL:               pgURL,
		RedisURL:            redisURL,
		PostCooldownSeconds: 0,
	}, true
}

// SpiritConfig stores configuration for the app.
type SpiritConfig struct {
	HTTPAddress         string
	CORSAllow           string
	PGURL               string
	RedisURL            string
	PostCooldownSeconds int
}

// ParseEnv parses system environment variables, returning app configuration.
func ParseEnv() *SpiritConfig {

	conf := &SpiritConfig{
		HTTPAddress:         "0.0.0.0:3000",
		CORSAllow:           "https://example.com",
		PGURL:               os.Getenv("SPIRITCHAT_PG_URL"),
		RedisURL:            os.Getenv("SPIRITCHAT_REDIS_URL"),
		PostCooldownSeconds: getPostCooldownEnv("SPIRITCHAT_COOLDOWN"),
	}
	if addr, ok := os.LookupEnv("SPIRITCHAT_ADDRESS"); ok {
		conf.HTTPAddress = addr
	}

	if allow, ok := os.LookupEnv("SPIRITCHAT_CORS_ALLOW"); ok {
		conf.CORSAllow = allow
	}
	return conf
}
