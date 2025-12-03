# Linux Dockerfile for aks-mcp
# Build stage
FROM golang:1.25-alpine AS builder
ARG TARGETOS=linux
ARG TARGETARCH
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

# Build the application for target platform with version injection
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -tags withoutebpf \
    -ldflags "-X github.com/Azure/aks-mcp/internal/version.GitVersion=${VERSION} \
              -X github.com/Azure/aks-mcp/internal/version.GitCommit=${GIT_COMMIT} \
              -X github.com/Azure/aks-mcp/internal/version.GitTreeState=${GIT_TREE_STATE} \
              -X github.com/Azure/aks-mcp/internal/version.BuildMetadata=${BUILD_DATE}" \
    -o aks-mcp ./cmd/aks-mcp

# Runtime stage
FROM alpine:3.22
ARG TARGETARCH

# Install required packages for kubectl and helm, plus build tools for Azure CLI
RUN apk add --no-cache curl bash openssl ca-certificates git python3 py3-pip \
    gcc python3-dev musl-dev linux-headers

# Install kubectl
RUN echo $TARGETARCH; curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/${TARGETARCH}/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/kubectl

# Install kubelogin
RUN KUBELOGIN_VERSION="v0.2.10" && \
    curl -LO "https://github.com/Azure/kubelogin/releases/download/${KUBELOGIN_VERSION}/kubelogin-linux-${TARGETARCH}.zip" && \
    unzip kubelogin-linux-${TARGETARCH}.zip && \
    mv bin/linux_${TARGETARCH}/kubelogin /usr/local/bin/kubelogin && \
    chmod +x /usr/local/bin/kubelogin && \
    rm -r bin/linux_${TARGETARCH} kubelogin-linux-${TARGETARCH}.zip

# Install helm
RUN HELM_ARCH=${TARGETARCH} && \
    curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && \
    chmod 700 get_helm.sh && \
    VERIFY_CHECKSUM=false ./get_helm.sh && \
    rm get_helm.sh

# Install Azure CLI
RUN pip3 install --break-system-packages --no-cache-dir azure-cli

# Install Cilium CLI
RUN CILIUM_VERSION="v0.18.9" && \
    CILIUM_ARCH=$([ "$TARGETARCH" = "amd64" ] && echo "amd64" || echo "arm64") && \
    curl -L --fail --remote-name-all "https://github.com/cilium/cilium-cli/releases/download/${CILIUM_VERSION}/cilium-linux-${CILIUM_ARCH}.tar.gz{,.sha256sum}" && \
    sha256sum -c cilium-linux-${CILIUM_ARCH}.tar.gz.sha256sum && \
    tar xzvf cilium-linux-${CILIUM_ARCH}.tar.gz && \
    mv cilium /usr/local/bin/cilium && \
    chmod +x /usr/local/bin/cilium && \
    rm cilium-linux-${CILIUM_ARCH}.tar.gz cilium-linux-${CILIUM_ARCH}.tar.gz.sha256sum

# Install Hubble CLI
RUN HUBBLE_VERSION="v1.18.3" && \
    HUBBLE_ARCH=$([ "$TARGETARCH" = "amd64" ] && echo "amd64" || echo "arm64") && \
    curl -L --fail --remote-name-all "https://github.com/cilium/hubble/releases/download/${HUBBLE_VERSION}/hubble-linux-${HUBBLE_ARCH}.tar.gz{,.sha256sum}" && \
    sha256sum -c hubble-linux-${HUBBLE_ARCH}.tar.gz.sha256sum && \
    tar xzvf hubble-linux-${HUBBLE_ARCH}.tar.gz && \
    mv hubble /usr/local/bin/hubble && \
    chmod +x /usr/local/bin/hubble && \
    rm hubble-linux-${HUBBLE_ARCH}.tar.gz hubble-linux-${HUBBLE_ARCH}.tar.gz.sha256sum

# Create the mcp user and group
RUN addgroup -S mcp && \
    adduser -S -G mcp -h /home/mcp mcp && \
    mkdir -p /home/mcp/.kube && \
    chown -R mcp:mcp /home/mcp

# Copy binary from builder
COPY --from=builder /app/aks-mcp /usr/local/bin/aks-mcp

# Set working directory
WORKDIR /home/mcp

# Expose the default port for sse/streamable-http transports
EXPOSE 8000

# Switch to non-root user
USER mcp

# Set environment variables
ENV HOME=/home/mcp \
    KUBECONFIG=/home/mcp/.kube/config

# Command to run
ENTRYPOINT ["/usr/local/bin/aks-mcp"]
CMD ["--transport", "streamable-http", "--host", "0.0.0.0"]
