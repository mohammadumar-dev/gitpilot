BINARY  := gitpilot
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-X main.version=$(VERSION)"

.PHONY: build fmt clean release-dry

build:
	go build $(LDFLAGS) -o $(BINARY) .

fmt:
	gofmt -w main.go

clean:
	rm -f $(BINARY)

release-dry:
	goreleaser release --snapshot --clean
