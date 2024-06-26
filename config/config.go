package config

import (
	"os"
)

/*
GetIntegrationsConfig is a testing function,
returns false if integrations shouldn't be run, or true, and integration config.
*/
func GetIntegrationsConfig() (*SpiritConfig, bool) {
	val, present := os.LookupEnv("SPIRIT_INTEGRATIONS")
	runIntegrations := false
	if present && len(val) > 0 && val != "0" && val != "FALSE" {
		runIntegrations = true
	}

	return ParseEnv(), runIntegrations
}

type SpiritAuthConfig struct {
	Domain       string
	ClientID     string
	ClientSecret string
}

func parseAuthEnv() SpiritAuthConfig {
	return SpiritAuthConfig{
		Domain:       os.Getenv("AUTH_DOMAIN"),
		ClientID:     os.Getenv("AUTH_CLIENTID"),
		ClientSecret: os.Getenv("AUTH_CLIENTSECRET"),
	}
}

// SpiritConfig stores configuration for the app.
type SpiritConfig struct {
	HTTPAddress string
	CORSAllow   string
	PGURL       string
	AuthConfig  SpiritAuthConfig
}

// ParseEnv parses system environment variables, returning app configuration.
func ParseEnv() *SpiritConfig {

	conf := &SpiritConfig{
		HTTPAddress: "0.0.0.0:3000",
		CORSAllow:   "https://example.com",
		PGURL:       os.Getenv("SPIRITCHAT_PG_URL"),
		AuthConfig:  parseAuthEnv(),
	}
	if addr, ok := os.LookupEnv("SPIRITCHAT_ADDRESS"); ok {
		conf.HTTPAddress = addr
	}

	if allow, ok := os.LookupEnv("SPIRITCHAT_CORS_ALLOW"); ok {
		conf.CORSAllow = allow
	}
	return conf
}
