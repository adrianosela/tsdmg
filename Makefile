SHELL := /bin/bash

# Helper function to check if TSDMG_TS_AUTHKEY is set
define check_ts_authkey
	@if [ -z "$$TSDMG_TS_AUTHKEY" ]; then \
		echo "ERROR: TSDMG_TS_AUTHKEY is not set. Please set it before running this command."; \
		exit 1; \
	fi
endef

# Helper function to check if TSDMG_CLOUDFLARE_API_TOKEN is set
define check_cloudflare_api_token
        @if [ -z "$$TSDMG_CLOUDFLARE_API_TOKEN" ]; then \
                echo "ERROR: TSDMG_CLOUDFLARE_API_TOKEN is not set. Please set it before running this command."; \
                exit 1; \
        fi
endef

# Helper function to check if TSDMG_GODADDY_API_TOKEN is set
define check_godaddy_api_token
        @if [ -z "$$TSDMG_GODADDY_API_TOKEN" ]; then \
                echo "ERROR: TSDMG_GODADDY_API_TOKEN is not set. Please set it before running this command."; \
                exit 1; \
        fi
endef

.PHONY: help
help: ## Print this help menu
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: tsdmg-cloudflare
tsdmg-cloudflare: ## Run the tsdmg server with Cloudflare as the DNS provider
	$(call check_ts_authkey)
	$(call check_cloudflare_api_token)
	go run ./cmd/server \
		-ts-authkey=$$TSDMG_TS_AUTHKEY \
		-dns-provider=cloudflare \
		-cloudflare-api-token=$$TSDMG_CLOUDFLARE_API_TOKEN

.PHONY: tsdmg-godaddy
tsdmg-godaddy: ## Run the tsdmg server with GoDaddy as the DNS provider
	$(call check_ts_authkey)
	$(call check_godaddy_api_token)
	go run ./cmd/server \
		-ts-authkey=$$TSDMG_TS_AUTHKEY \
		-dns-provider=godaddy \
		-godaddy-api-token=$$TSDMG_GODADDY_API_TOKEN

.PHONY: lint
lint: ## Lint code
	@golangci-lint run ./...
