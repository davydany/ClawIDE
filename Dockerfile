FROM golang:1.24-alpine AS builder

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN apk add --no-cache git nodejs npm

WORKDIR /app

# Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Node dependencies for frontend build
COPY web/src/package.json web/src/package-lock.json* web/src/
RUN cd web/src && npm install

# Copy source
COPY . .

# Build frontend assets (xterm + codemirror bundles)
RUN cd web/src && npm run build
RUN npx tailwindcss@3 -i web/static/css/input.css -o web/static/dist/app.css --minify

# Build Go binary with version injection
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/davydany/ClawIDE/internal/version.Version=${VERSION} -X github.com/davydany/ClawIDE/internal/version.Commit=${COMMIT} -X github.com/davydany/ClawIDE/internal/version.Date=${DATE}" \
    -o /clawide ./cmd/clawide

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache bash git docker-cli docker-cli-compose curl

COPY --from=builder /clawide /usr/local/bin/clawide

EXPOSE 9800

ENTRYPOINT ["clawide"]
