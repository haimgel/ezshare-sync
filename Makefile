.PHONY: all test lint build release clean

BINARY_NAME=ezshare-sync
DIST_DIR=dist
CMD_PATH=./cmd/ezshare-sync

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

clean:
	rm -rf $(DIST_DIR)
	go clean