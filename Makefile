SHELL := /bin/bash

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
tsdmg: ## Build the tsdmg server binary for the current OS/ARCH
	(cd cmd/server && go build -o $(CURDIR)/tsdmg)

.PHONY: tsdmg-run
tsdmg-run: ## Run the tsdmg server (with Cloudflare as the DNS provider)
	$(call check_env_set,TSDMG_TS_AUTHKEY)
	$(call check_env_set,TSDMG_CLOUDFLARE_API_TOKEN)
	(cd cmd/server && go run . \
		-domain=tsdmg.net \
		-dns-provider=cloudflare \
		-cloudflare-api-token=$$TSDMG_CLOUDFLARE_API_TOKEN \
		-ts-authkey=$$TSDMG_TS_AUTHKEY)

.PHONY: tsdmg-image
tsdmg-image: ## Build tsdmg server Docker image
	docker build \
		-f Dockerfile.tsdmg \
		-t ghcr.io/adrianosela/tsdmg:latest \
		.

.PHONY: tsdmg-image-run
tsdmg-image-run: ## Run the tsdmg server image (with Cloudflare as the DNS provider)
	$(call check_env_set,TSDMG_TS_AUTHKEY)
	$(call check_env_set,TSDMG_CLOUDFLARE_API_TOKEN)
	docker run -d -it ghcr.io/adrianosela/tsdmg \
		-domain=tsdmg.net \
		-dns-provider=cloudflare \
		-cloudflare-api-token=$$TSDMG_CLOUDFLARE_API_TOKEN \
		-ts-authkey=$$TSDMG_TS_AUTHKEY

.PHONY: tsdmg-image-push
tsdmg-image-push: ## Push tsdmg server Docker image to registry
	docker push ghcr.io/adrianosela/tsdmg:latest .

.PHONY: tsautocert
tsautocert: ## Build the tsautocert binary for the current OS/ARCH
	(cd cmd/tsautocert && go build -o $(CURDIR)/tsautocert)

.PHONY: tsautocert-image
tsautocert-image: ## Build tsautocert Docker image
	docker build \
		-f Dockerfile.tsautocert \
		-t ghcr.io/adrianosela/tsautocert:latest \
		.

.PHONY: tsautocert-image-run
tsautocert-image-run: ## Run the tsautocert Docker image
	@set -e; \
	addr="$$(dscacheutil -q host -a name tsdmg | awk '/ip_address:/{print $$2; exit}')"; \
	if [ -z "$$addr" ]; then echo "Could not resolve tsdmg via dscacheutil"; exit 1; fi; \
	case "$$addr" in *:*) addr="[$$addr]";; esac; \
	docker run -d -v "$(CURDIR)/certcache:/root/certcache" -p 443:443 ghcr.io/adrianosela/tsautocert \
		-server-url="http://$$addr" \
		-cn="adrianos-laptop.tsdmg.net" \
		-cachedir=/root/certcache \
		-ensure-address-records \
		-serve-hello-world

.PHONY: tsautocert-image-push
tsautocert-image-push: ## Push tsautocert Docker image to registry
	docker push ghcr.io/adrianosela/tsautocert:latest

.PHONY: lint
lint: ## Lint code
	@golangci-lint run ./...
