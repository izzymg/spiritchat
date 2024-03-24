# SpiritChat

SpiritChat is a chan-style imageboard/BBS written in Golang.

Frontend (Vue.JS): https://github.com/izzymg/spiritclient

### Usage
`spirit` - start spirit

`spirit migrate up` - apply migrations up

`spirit migrate down` - drops everything

### devcontainer

Developed inside vscode devcontainer with Redis & Postgres.

### Environment variables

`SPIRITCHAT_PG_URL` `SPIRITCHAT_REDIS_URL` `SPIRITCHAT_ADDRESS` `SPIRITCHAT_CORS_ALLOW`


#### Integration tests

Set `SPIRIT_INTEGRATIONS` if you want integration tests.
