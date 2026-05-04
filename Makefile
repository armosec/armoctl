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

.PHONY: skill-docs verify-skill-docs

# Regenerate per-cluster skill markdown files.
skill-docs:
	go run ./cmd/gen-skill-docs

# CI gate — fail if generated skills are stale relative to source.
verify-skill-docs:
	@$(MAKE) skill-docs
	@if ! git diff --quiet -- skills/; then \
		echo "ERROR: skills/ is stale. Run 'make skill-docs' and commit the result."; \
		git --no-pager diff -- skills/; \
		exit 1; \
	fi
