FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git nodejs npm

WORKDIR /app

# Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Node dependencies for frontend build
COPY web/src/package.json web/src/
RUN cd web/src && npm install

# Copy source
COPY . .

# Build frontend assets
RUN cd web/src && npx esbuild xterm-entry.js --bundle --minify --outfile=../static/dist/xterm-bundle.js
RUN npx tailwindcss@3 -i web/static/css/input.css -o web/static/dist/app.css --minify

# Build Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /clawide ./cmd/clawide

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache bash git docker-cli docker-cli-compose curl

COPY --from=builder /clawide /usr/local/bin/clawide

EXPOSE 9800

ENTRYPOINT ["clawide"]
