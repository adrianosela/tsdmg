SHELL := /bin/bash
PROGRAM_NAME := tsdmg
DOCKER_IMAGE := ghcr.io/adrianosela/$(PROGRAM_NAME):latest

define check_env_set
	@if [ -z "$$$(1)" ]; then \
		echo "ERROR: $(1) is not set. Please set it before running this command."; \
		exit 1; \
	fi
endef

.PHONY: help
help: ## Print this help menu
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: tsdmg-cloudflare
tsdmg-cloudflare: ## Run the tsdmg server with Cloudflare as the DNS provider
	$(call check_env_set,TSDMG_TS_AUTHKEY)
	$(call check_env_set,TSDMG_CLOUDFLARE_API_TOKEN)
	go run ./cmd/server \
		-ts-authkey=$$TSDMG_TS_AUTHKEY \
		-dns-provider=cloudflare \
		-cloudflare-api-token=$$TSDMG_CLOUDFLARE_API_TOKEN

.PHONY: tsdmg-godaddy
tsdmg-godaddy: ## Run the tsdmg server with GoDaddy as the DNS provider
	$(call check_env_set,TSDMG_TS_AUTHKEY)
	$(call check_env_set,TSDMG_GODADDY_API_TOKEN)
	go run ./cmd/server \
		-ts-authkey=$$TSDMG_TS_AUTHKEY \
		-dns-provider=godaddy \
		-godaddy-api-token=$$TSDMG_GODADDY_API_TOKEN

.PHONY: build
build: ## Build the tsdmg server binary for the current OS/ARCH
	go build -o $(PROGRAM_NAME) ./cmd/server

.PHONY: image
image: ## Build tsdmg Docker image
	docker build -t $(DOCKER_IMAGE) .

.PHONY: lint
lint: ## Lint code
	@golangci-lint run ./...
