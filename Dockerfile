# Build stage
FROM golang:1.24-alpine AS builder
ARG VERSION
ARG GIT_COMMIT
ARG BUILD_DATE
ARG GIT_TREE_STATE
# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o aks-mcp ./cmd/aks-mcp

# Runtime stage
FROM alpine:3.21

# Install required packages for kubectl and helm
RUN apk add --no-cache curl bash openssl ca-certificates git

# Create the mcp user and group
RUN addgroup -S mcp && \
    adduser -S -G mcp -h /home/mcp mcp && \
    mkdir -p /home/mcp/.kube && \
    chown -R mcp:mcp /home/mcp

# Copy binary from builder
COPY --from=builder /app/aks-mcp /usr/local/bin/aks-mcp

# Set working directory
WORKDIR /home/mcp

# Switch to non-root user
USER mcp

# Set environment variables
ENV HOME=/home/mcp

# Command to run
ENTRYPOINT ["/usr/local/bin/aks-mcp"]
CMD ["--transport", "stdio"]
