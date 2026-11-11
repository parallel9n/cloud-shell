BINARY := cloud-shell
PREFIX ?= $(HOME)/.local
VERSION ?= dev

.PHONY: build test install fmt dist

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) .

test:
	go test ./...

fmt:
	gofmt -w .

install: build
	mkdir -p $(PREFIX)/bin
	install -m 755 $(BINARY) $(PREFIX)/bin/$(BINARY)

dist:
	bash scripts/release-assets.sh $(VERSION) dist
