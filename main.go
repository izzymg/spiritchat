package main

import (
	"context"
	"log"
	"os"
	"spiritchat/data"
	"spiritchat/serve"
)

const dbURL = "postgres://postgres:ferret@localhost:5432/spiritchat"

type config struct {
	HTTPAddress string
	CORSAllow   string
}

// Parse environment for configuration
func parseEnv() config {
	conf := config{
		HTTPAddress: "0.0.0.0:3000",
		CORSAllow:   "https://example.com",
	}
	if addr, ok := os.LookupEnv("SPIRITCHAT_ADDRESS"); ok {
		conf.HTTPAddress = addr
	}

	if allow, ok := os.LookupEnv("SPIRITCHAT_CORS_ALLOW"); ok {
		conf.CORSAllow = allow
	}
	return conf
}

// SpiritChat entry point
func main() {

	conf := parseEnv()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Establishing database connection")
	store, err := data.NewDatastore(ctx, dbURL)
	if err != nil {
		log.Printf("Failed to initalize database: %s", err)
	}

	server := serve.NewServer(store, conf.HTTPAddress)
	log.Printf("Starting server on %s, allowing %s CORS", conf.HTTPAddress, conf.CORSAllow)
	log.Println(server.Listen(ctx))
}
