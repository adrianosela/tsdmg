PROGRAM_NAME := tssl

.PHONY: build
build: ## Build CLI binary for current OS/ARCH and move to binaries path
	@go build -o $(PROGRAM_NAME) .
	@mv $(PROGRAM_NAME) /usr/local/bin/$(PROGRAM_NAME)
