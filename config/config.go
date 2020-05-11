package config

import "os"

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
