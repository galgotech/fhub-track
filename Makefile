
.PHONY: build

GO = go

build: ## Build client
	@echo "build client"
	$(GO) build -o ./bin/fhub-track ./cmd/fhub-track
