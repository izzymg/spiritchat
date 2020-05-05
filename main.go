package main

import (
	"context"
	"flag"
	"log"
	"spiritchat/data"
	"spiritchat/serve"
)

const dbURL = "postgres://postgres:ferret@localhost:5432/spiritchat"

func main() {

	serverAddress := flag.String("address", "0.0.0.0:3000", "HTTP Server address")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Establishing database connection")
	store, err := data.NewDatastore(ctx, dbURL)
	if err != nil {
		log.Printf("Failed to initalize database: %s", err)
	}

	server := serve.NewServer(store, *serverAddress)
	log.Printf("Starting server on %s", *serverAddress)
	log.Println(server.Listen(ctx))
}
