.PHONY: all test lint build release clean publish-cloudsmith

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c

BINARY_NAME=ezshare-sync
DIST_DIR=dist
CMD_PATH=./cmd/ezshare-sync
CLOUDSMITH_REPO=haimgel/public

all: test lint build

test:
	go test ./ezshare/... -v

lint:
	golangci-lint run

build:
	@mkdir -p $(DIST_DIR)
	go build -o $(DIST_DIR)/$(BINARY_NAME) $(CMD_PATH)

release:
	@mkdir -p $(DIST_DIR)
	goreleaser release --snapshot --clean

publish-cloudsmith:
	@echo "Publishing packages to Cloudsmith..."
	@for file in $(DIST_DIR)/*.deb; do \
		if [ -f "$$file" ]; then \
			echo "Uploading $$file to Cloudsmith (deb)..."; \
			cloudsmith push deb $(CLOUDSMITH_REPO)/any-distro/any-version "$$file"; \
		fi \
	done
	@for file in $(DIST_DIR)/*.rpm; do \
		if [ -f "$$file" ]; then \
			echo "Uploading $$file to Cloudsmith (rpm)..."; \
			cloudsmith push rpm $(CLOUDSMITH_REPO)/any-distro/any-version "$$file"; \
		fi \
	done
	@for file in $(DIST_DIR)/*.apk; do \
		if [ -f "$$file" ]; then \
			echo "Uploading $$file to Cloudsmith (alpine)..."; \
			cloudsmith push alpine $(CLOUDSMITH_REPO)/alpine/any-version "$$file"; \
		fi \
	done
	@echo "Cloudsmith publishing complete!"

clean:
	rm -rf $(DIST_DIR)
	go clean