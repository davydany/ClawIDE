.PHONY: build dev clean vendor-js css css-watch run test

# Binary name
BINARY := ccmux
BUILD_DIR := .

# Go build flags
LDFLAGS := -s -w

# Default target
all: vendor-js css build

# Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/ccmux

# Run in development mode
dev: vendor-js css
	go run ./cmd/ccmux

# Run the built binary
run: build
	./$(BINARY)

# Build Tailwind CSS
css:
	npx tailwindcss -i web/static/css/input.css -o web/static/dist/app.css --minify

# Watch Tailwind CSS for changes
css-watch:
	npx tailwindcss -i web/static/css/input.css -o web/static/dist/app.css --watch

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
	@cd web/src && npm install && npm run build

# Clean build artifacts
clean:
	rm -f $(BUILD_DIR)/$(BINARY)
	rm -f web/static/dist/app.css

# Run tests
test:
	go test ./...

tests: test

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

# Format code
fmt:
	go fmt ./...
