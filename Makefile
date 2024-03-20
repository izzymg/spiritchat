# TESTING
.PHONY: test test-integrate migrate-up migrate-down
test:
	@echo "--- Spirit Test: No Integrations ---"
	SPIRIT_INTEGRATIONS="" go test -cover -timeout 15s ./...
migrate-up:
	@echo "--- Migrating Spirit Up ---"
	go run . migrate up
migrate-down:
	@echo "--- Migrating Spirit Down ---"
	go run . migrate down
test-integrate:
	@echo "--- Spirit Test: With Integrations ---"
	SPIRIT_INTEGRATIONS="YES" go test -cover -timeout 15s ./...

