# SpiritChat

SpiritChat is a chan-style imageboard/BBS written in Golang.

Frontend (Vue.JS): https://github.com/izzymg/spiritclient

### Deploying

Using Docker is the easiest way to deploy SpiritChat.

Create Postgres and Redis containers, and link them both to the same Docker network.

TODO: Create a docker compose yml?

```sh
docker network create spirit

docker run --name spirit-pg --network spirit ... -d postgres

docker run --name spirit-redis --network spirit ... -d redis

```

Then use [db/migrate.sql](db.db/migrate.sql) to generate the tables and procedures.

TODO: Write a script for migrations?

Build Spirit's docker image and run it, passing in the URLs to connect to those instances.

```sh
docker build -t spirit .

docker run --name spiritchat \
--network spirit \
-e "SPIRITCHAT_PG_URL=postgres://postgres:password@spirit-pg/spiritchat" \
-e "SPIRITCHAT_REDIS_URL=redis://someusr:agoodpassword@spirit-redis" \
-e "SPIRITCHAT_ADDRESS=0.0.0.0:3000" \
-e "SPIRITCHAT_CORS_ALLOW=https://yoursite.com" \
-p "3000:3000" \
... spirit
```


### Integration tests

You'll need to set the env var `SPIRITTEST_INTEGRATIONS` to run integration tests. 

They also use different environment variables to connect to the data stores so you run no risk of running them in prod. They setup and tear down tables.

Use for Postgres, Redis, and HTTP respectively:

`SPIRITTEST_PG_URL` `SPIRITTEST_REDIS_URL` `SPIRITTEST_ADDR`


#### Design

[Design docs](docs/design.md)
