package main

import (
	"context"
	"log"
	"spiritchat/config"
	"spiritchat/data"
	"spiritchat/serve"
)

func main() {

	conf := config.ParseEnv()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Establishing database connection")
	store, err := data.NewDatastore(ctx, conf.PGURL, conf.RedisURL)
	if err != nil {
		log.Printf("Failed to initalize database: %s", err)
		return
	}
	defer store.Cleanup(ctx)

	server := serve.NewServer(store, serve.ServerOptions{
		Address:             conf.HTTPAddress,
		CorsOriginAllow:     conf.CORSAllow,
		PostCooldownSeconds: 30,
	})
	log.Printf("Starting server on %s, allowing %s CORS", conf.HTTPAddress, conf.CORSAllow)
	log.Println(server.Listen(ctx))
}
