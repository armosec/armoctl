VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
ARCH?=$(shell go env GOARCH)

.PHONY: armoctl

armoctl:
	CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(ARCH) go build \
		-ldflags "-X main.Version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)" \
		-o armoctl .

.PHONY: schemas
schemas:
	./scripts/gen-schemas.sh
