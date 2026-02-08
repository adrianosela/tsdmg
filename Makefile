SHELL := /bin/bash
SERVER_PROGRAM_NAME := tsdmg 
CLI_PROGRAM_NAME := tssl

# Helper function to check if TSDMG_TS_AUTHKEY is set
define check_ts_authkey
	@if [ -z "$$TSDMG_TS_AUTHKEY" ]; then \
		echo "ERROR: TSDMG_TS_AUTHKEY is not set. Please set it before running this command."; \
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

.PHONY: tsdmg-godadd
tsdmg-godaddy: ## Run the tsdmg server with GoDaddy as the DNS provider
	$(call check_ts_authkey)
	$(call check_godaddy_api_token)
	go run ./cmd/server-examples/godaddy \
		-ts-authkey=$$TSDMG_TS_AUTHKEY \
		-godaddy-api-token=$$TSDMG_GODADDY_API_TOKEN

.PHONY: cli
cli: ## Build CLI binary for current OS/ARCH and move to binaries path
	@go build -o $(CLI_PROGRAM_NAME) ./cmd/cli/main.go
	@mv $(CLI_PROGRAM_NAME) /usr/local/bin/$(CLI_PROGRAM_NAME)

.PHONY: lint
lint: ## Lint code
	@golangci-lint run ./...
