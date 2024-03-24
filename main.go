package main

import (
	"context"
	"log"
	"os"
	"spiritchat/auth"
	"spiritchat/config"
	"spiritchat/data"
	"spiritchat/serve"
)

func isMigration() bool {
	return len(os.Args) > 2 && os.Args[1] == "migrate" && (os.Args[2] == "up" || os.Args[2] == "down")
}

// true = up false = down
func getMigrationType() bool {
	return os.Args[2] == "up"
}

func main() {
	conf := config.ParseEnv()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Establishing database connection")
	store, err := data.NewDatastore(ctx, conf.PGURL, 15)
	if err != nil {
		log.Fatalf("Failed to initalize database: %+v", err)
		return
	}
	defer store.Cleanup(ctx)

	if isMigration() {
		migrationType := getMigrationType()
		if migrationType {
			log.Println("Migrating up")
		} else {
			log.Println("Migrating down")
		}
		err := store.Migrate(ctx, migrationType)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("Establishing OAuth API")
		auth, err := auth.NewOAuth(ctx, conf.AuthConfig)
		if err != nil {
			log.Fatalf("Failed to initialize OAuth API: %+v", err)
			return
		}
		server := serve.NewServer(store, auth, serve.ServerOptions{
			Address:             conf.HTTPAddress,
			CorsOriginAllow:     conf.CORSAllow,
			PostCooldownSeconds: conf.PostCooldownSeconds,
		})
		log.Printf("Starting server on %s, allowing %s CORS", conf.HTTPAddress, conf.CORSAllow)
		log.Println(server.Listen(ctx))
	}
}
