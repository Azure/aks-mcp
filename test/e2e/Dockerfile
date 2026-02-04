FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the test binary
RUN CGO_ENABLED=0 GOOS=linux go build -o e2e-test ./cmd/e2e-test

# Final stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates curl

COPY --from=builder /app/e2e-test /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/e2e-test"]
