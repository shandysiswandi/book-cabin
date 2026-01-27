.PHONY: run
run:
	@LOCAL=true go run main.go

.PHONY: lint
lint:
	@golangci-lint run
