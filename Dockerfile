# file: Dockerfile
# Optimized multi-stage Dockerfile to reduce build times from ~20min to ~3-5min

# Stage 1: Node.js build stage (can be cached separately)
FROM node:26-bookworm AS node-builder
WORKDIR /src/webui

# Copy package files first for better caching
COPY webui/package*.json ./

# Clear npm cache and ensure clean install with pinned esbuild version
RUN npm cache clean --force && \
    rm -rf node_modules package-lock.json && \
    npm install --legacy-peer-deps --production=false

# Copy source and build
COPY webui/ ./
RUN npm run build

# Stage 2: Go dependencies (can be cached separately)
FROM golang:1.26.5-bookworm AS go-deps
WORKDIR /src

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    gcc \
    sqlite3 libsqlite3-dev

# Copy go.mod and go.sum first for better dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Stage 3: Go build stage
FROM go-deps AS go-builder
WORKDIR /src

# Version information build arguments
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

# Set environment variable to indicate Docker build
ENV DOCKER_BUILD=1

# Copy source code
COPY . .

# Copy the built web assets from node-builder stage
COPY --from=node-builder /src/webui/dist ./webui/dist

# Note: No need to run 'go generate ./webui' since we already have the dist directory
# The embed.go file will automatically embed the dist directory during go build

# Build the application with version information
# Use conditional CGO and build tags based on target architecture
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "arm64" ]; then \
    echo "Building for ARM64 with pure Go (no CGO)" && \
    CGO_ENABLED=0 go build -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${GIT_COMMIT}'" -o subtitle-manager ./; \
    else \
    echo "Building for AMD64 with CGO and SQLite support" && \
    CGO_ENABLED=1 go build -tags=sqlite -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${GIT_COMMIT}'" -o subtitle-manager ./; \
    fi

# Stage 4: Final runtime image
FROM golang:1.26.5-bookworm AS final

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    ca-certificates \
    tzdata

# Explicitly set the ffmpeg path for the application
ENV SM_FFMPEG_PATH=/usr/bin/ffmpeg

# Create non-root user
RUN addgroup --system subtitle && adduser --system --ingroup subtitle subtitle

# Create directories
RUN mkdir -p /config /media && \
    chown -R subtitle:subtitle /config /media

# Copy binary
COPY --from=go-builder /src/subtitle-manager /usr/local/bin/subtitle-manager
RUN chmod +x /usr/local/bin/subtitle-manager
# Copy docker init script
COPY docker-init.sh /usr/local/bin/docker-init.sh
RUN chmod +x /usr/local/bin/docker-init.sh

# --- Security update step (close to the end for best caching) ---
RUN apt-get update && apt-get full-upgrade -y && apt-get autoremove -y && apt-get clean

# Switch to non-root user
USER subtitle

# Set default environment variables
ENV SM_CONFIG_FILE=/config/subtitle-manager.yaml \
    SM_DB_PATH=/config/db \
    SM_DB_BACKEND=pebble \
    SM_SQLITE3_FILENAME=subtitle-manager.db \
    SM_LOG_LEVEL=info

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-init.sh"]
CMD ["web"]
