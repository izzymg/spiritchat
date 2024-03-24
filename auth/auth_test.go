package auth

import (
	"context"
	"spiritchat/config"
	"testing"
)

func createExampleAuthConfig() config.SpiritAuthConfig {
	return config.SpiritAuthConfig{
		Domain:       "example.us.auth0.com",
		ClientID:     "EXAMPLE_16L9d34h0qe4NVE6SaHxZEid",
		ClientSecret: "EXAMPLE_XSQGmnt8JdXs23407hrK6XXXXXXX",
	}
}

func TestNew(t *testing.T) {
	_, err := NewOAuth(context.TODO(), createExampleAuthConfig())
	if err != nil {
		t.Errorf("auth client couldn't be created: %v", err)
	}
}
