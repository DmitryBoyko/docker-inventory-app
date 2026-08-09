VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)
DIST_DIR := internal/uiembed/dist

.PHONY: tidy test test-integration parity ui sync-ui build build-all run release-snapshot clean

tidy:
	go mod tidy

test:
	go test ./...

test-integration:
	go test -tags=integration ./internal/docker/ -count=1 -v

parity:
	go run ./cmd/parity-check -skip-stats

# Build Vite app into web/dist and sync into the embed package.
ui: sync-ui

sync-ui:
	cd web && npm ci && npm run build
	rm -rf "$(DIST_DIR)"
	mkdir -p "$(DIST_DIR)"
	cp -R web/dist/. "$(DIST_DIR)/"

build: sync-ui
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer$(shell go env GOEXE) ./cmd/docker-visualizer

# Cross-compile without re-running npm (expects sync-ui already done).
build-all:
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer-windows-amd64.exe ./cmd/docker-visualizer
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer-linux-amd64 ./cmd/docker-visualizer
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer-linux-arm64 ./cmd/docker-visualizer
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer-darwin-amd64 ./cmd/docker-visualizer
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/docker-visualizer-darwin-arm64 ./cmd/docker-visualizer
	cd bin && sha256sum docker-visualizer-* > SHA256SUMS 2>/dev/null || shasum -a 256 docker-visualizer-* > SHA256SUMS

run:
	go run ./cmd/docker-visualizer

release-snapshot: sync-ui
	goreleaser release --snapshot --clean

clean:
	rm -rf bin/ dist/ web/dist/
	# restore embed stub after clean of synced assets
	mkdir -p "$(DIST_DIR)"
	@test -f "$(DIST_DIR)/index.html" || true
