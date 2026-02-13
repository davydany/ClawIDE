.PHONY: build dev clean vendor-js css css-watch run test version

# Binary name
BINARY := clawide
BUILD_DIR := .

# Version injection
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PKG     := github.com/davydany/ClawIDE/internal/version

# Go build flags
LDFLAGS := -s -w -X $(PKG).Version=$(VERSION) -X $(PKG).Commit=$(COMMIT) -X $(PKG).Date=$(DATE)

# Default target
all: vendor-js css build

# Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/clawide

# Run in development mode
dev: vendor-js css
	go run ./cmd/clawide

# Run the built binary
run: build
	./$(BINARY)

# Build Tailwind CSS
css:
	./web/src/node_modules/.bin/tailwindcss -i web/static/css/input.css -o web/static/dist/app.css --minify

# Watch Tailwind CSS for changes
css-watch:
	./web/src/node_modules/.bin/tailwindcss -i web/static/css/input.css -o web/static/dist/app.css --watch

# Download and vendor JS dependencies
vendor-js:
	@mkdir -p web/static/vendor
	@if [ ! -f web/static/vendor/htmx.min.js ]; then \
		curl -sL https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js -o web/static/vendor/htmx.min.js; \
		echo "Downloaded htmx.min.js"; \
	fi
	@if [ ! -f web/static/vendor/alpine.min.js ]; then \
		curl -sL https://unpkg.com/alpinejs@3.14.8/dist/cdn.min.js -o web/static/vendor/alpine.min.js; \
		echo "Downloaded alpine.min.js"; \
	fi
	@if [ ! -f web/static/vendor/qrcode.min.js ]; then \
		curl -sL https://cdnjs.cloudflare.com/ajax/libs/qrcode-generator/1.4.4/qrcode.min.js -o web/static/vendor/qrcode.min.js; \
		echo "Downloaded qrcode.min.js"; \
	fi
	@cd web/src && npm install && npm run build

# Clean build artifacts
clean:
	rm -f $(BUILD_DIR)/$(BINARY)
	rm -f web/static/dist/app.css

# Run tests
test:
	go test -race ./...

tests: test

# Run tests with coverage report
coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Docker compose commands
start:
	docker compose up -d

up: start

stop:
	docker compose down

shutdown: stop

status:
	docker compose ps -a

ps: status

logs:
ifdef SERVICE
	docker compose logs -f $(SERVICE)
else
	docker compose logs -f
endif

# Print version info (for debugging)
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

# Format code
fmt:
	go fmt ./...

# ---- Documentation Site ----
.PHONY: docs docs-dev docs-deps docs-screenshots docs-clean

docs:
	cd docs-website && hugo --minify --gc

docs-dev:
	cd docs-website && hugo server --buildDrafts --navigateToChanged

docs-deps:
	cd docs-website && npm install

docs-screenshots:
	cd docs-website && node scripts/capture-screenshots.js

docs-clean:
	rm -rf docs-website/public docs-website/resources
