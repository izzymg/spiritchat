# TESTING
.PHONY: test test-integrate migrate-up migrate-down
test:
	@echo "--- Spirit Test: No Integrations ---"
	go run . migrate up
	SPIRIT_INTEGRATIONS="" go test -count=1 -cover -timeout 15s ./...
migrate-up:
	@echo "--- Migrating Spirit Up ---"
	go run . migrate up
migrate-down:
	@echo "--- Migrating Spirit Down ---"
	go run . migrate down
test-integrations:
	@echo "--- Spirit Test: With Integrations ---"
	go run . migrate up
	SPIRIT_INTEGRATIONS="YES" go test -count=1 -cover -timeout 15s ./...

