# SpiritChat

SpiritChat is a chan-style imageboard/BBS written in Golang.

Frontend (Vue.JS): https://github.com/izzymg/spiritclient

### Usage
`spirit` - start spirit

`spirit migrate up` - apply migrations up

`spirit migrate down` - drops everything

### devcontainer

Developed inside vscode devcontainer with Redis & Postgres.

Use `launch.json` to run, and run up/down migration tests

### Environment variables

`SPIRITCHAT_PG_URL` `SPIRITCHAT_REDIS_URL` `SPIRITCHAT_ADDR`


#### Integration tests

Set the env var `SPIRITTEST_INTEGRATIONS` to run integration tests, and then set:

`SPIRITTEST_PG_URL` `SPIRITTEST_REDIS_URL` `SPIRITTEST_ADDR`


