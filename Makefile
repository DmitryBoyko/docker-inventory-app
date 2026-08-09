VERSION_FILE := VERSION
VERSION ?= $(shell tr -d ' \r\n' < $(VERSION_FILE) 2>/dev/null || echo 0.1.0)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)
DIST_DIR := internal/uiembed/dist

.PHONY: tidy test test-integration parity ui sync-ui bump-version build cross build-all run release-snapshot clean

tidy:
	go mod tidy

test:
	go test ./...

test-integration:
	go test -tags=integration ./internal/docker/ -count=1 -v

parity:
	go run ./cmd/parity-check -skip-stats

bump-version:
	bash scripts/bump-version.sh

# Build Vite app into web/dist and sync into the embed package.
ui: sync-ui

sync-ui:
	cd web && npm ci && npm run build
	rm -rf "$(DIST_DIR)"
	mkdir -p "$(DIST_DIR)"
	cp -R web/dist/. "$(DIST_DIR)/"
	@test -f "$(DIST_DIR)/index.html"

# Host binary — bump patch, refresh UI, stamp SemVer into binary.
build: bump-version sync-ui
	$(eval VERSION := $(shell tr -d ' \r\n' < $(VERSION_FILE)))
	$(eval LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT))
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer$(shell go env GOEXE) ./cmd/docker-visualizer
	@echo "Built v$(VERSION)"

# Cross-compile only (assumes embed already synced). Prefer build-all for releases.
cross:
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer-windows-amd64.exe ./cmd/docker-visualizer
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer-linux-amd64 ./cmd/docker-visualizer
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer-linux-arm64 ./cmd/docker-visualizer
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer-darwin-amd64 ./cmd/docker-visualizer
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer-darwin-arm64 ./cmd/docker-visualizer
	cd bin && sha256sum docker-visualizer-* > SHA256SUMS 2>/dev/null || shasum -a 256 docker-visualizer-* > SHA256SUMS

# Full release matrix: bump + UI sync + all platforms.
build-all: bump-version sync-ui
	$(eval VERSION := $(shell tr -d ' \r\n' < $(VERSION_FILE)))
	$(eval LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT))
	$(MAKE) cross VERSION=$(VERSION) LDFLAGS="$(LDFLAGS)"

run:
	go run -ldflags "$(LDFLAGS)" ./cmd/docker-visualizer

release-snapshot: bump-version sync-ui
	goreleaser release --snapshot --clean

clean:
	rm -rf bin/ dist/ web/dist/
	mkdir -p "$(DIST_DIR)"
	@test -f "$(DIST_DIR)/index.html" || true
