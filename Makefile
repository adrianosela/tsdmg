SHELL := /bin/bash
PROGRAM_NAME := tsdmg
DOCKER_IMAGE := ghcr.io/adrianosela/$(PROGRAM_NAME):latest
TSDMG_DOMAIN := tsdmg.net

define check_env_set
	@if [ -z "$$$(1)" ]; then \
		echo "ERROR: $(1) is not set. Please set it before running this command."; \
		exit 1; \
	fi
endef

.PHONY: help
help: ## Print this help menu
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: tsdmg
tsdmg: ## Run the tsdmg server (with Cloudflare as the DNS provider)
	$(call check_env_set,TSDMG_TS_AUTHKEY)
	$(call check_env_set,TSDMG_CLOUDFLARE_API_TOKEN)
	(cd cmd/server && go run . \
		-ts-authkey=$$TSDMG_TS_AUTHKEY \
		-dns-provider=cloudflare \
		-cloudflare-api-token=$$TSDMG_CLOUDFLARE_API_TOKEN \
		-domain=$(TSDMG_DOMAIN) \
		-node-reg-domain=$(TSDMG_DOMAIN))

.PHONY: tsdmg-docker
tsdmg-docker: ## Run the tsdmg server image (with Cloudflare as the DNS provider)
	$(call check_env_set,TSDMG_TS_AUTHKEY)
	$(call check_env_set,TSDMG_CLOUDFLARE_API_TOKEN)
	docker run -d -it ghcr.io/adrianosela/tsdmg \
		-ts-authkey=$$TSDMG_TS_AUTHKEY \
		-dns-provider=cloudflare \
		-cloudflare-api-token=$$TSDMG_CLOUDFLARE_API_TOKEN \
		-domain=$(TSDMG_DOMAIN) \
		-node-reg-domain=$(TSDMG_DOMAIN)

.PHONY: build
build: ## Build the tsdmg server binary for the current OS/ARCH
	(cd cmd/server && go build -o $(CURDIR)/$(PROGRAM_NAME))

.PHONY: image
image: ## Build tsdmg Docker image
	docker build -t $(DOCKER_IMAGE) .

.PHONY: push-image
push-image: ## Push tsdmg Docker image to registry
	docker push $(DOCKER_IMAGE)

.PHONY: lint
lint: ## Lint code
	@golangci-lint run ./...
