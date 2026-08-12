<!-- file: .devcontainer/README.md -->
<!-- version: 1.0.0 -->
<!-- guid: 2a352a51-01b6-4731-9cb6-fedc6932c523 -->
<!-- last-edited: 2026-08-12 -->

# Development Container

This directory contains the development container configuration for Subtitle
Manager. The dev container provides a consistent development environment with
all necessary tools and dependencies pre-installed.

## Features

- **Go 1.24+** with development tools (golangci-lint, goimports, dlv, etc.)
- **Node.js 20** for React frontend development (installed via apt-get)
- **FFmpeg** for subtitle processing
- **SQLite** with CGO support enabled
- **VS Code extensions** for Go and React development
- **Pre-commit hooks** automatically installed
- **Port forwarding** for web UI (3000) and application (8080)

## Prerequisites

- [Visual Studio Code](https://code.visualstudio.com/)
- [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
- [Docker Desktop](https://www.docker.com/products/docker-desktop) or Docker
  Engine

## Getting Started

1. Clone the repository
2. Open the repository in VS Code
3. When prompted, click "Reopen in Container" or run the command palette:
   `Dev Containers: Reopen in Container`
4. Wait for the container to build and the post-create script to run
5. Start developing!

## Development Workflow

### Backend Development

```bash
# Build the application
make build

# Run tests with SQLite support (recommended for dev container)
make test-sqlite

# Run tests with race detection
make test-race-sqlite

# Start development server
make dev
```

### Frontend Development

```bash
# Navigate to web UI directory
cd webui

# Start development server with hot reloading
npm run dev

# Run tests
npm test

# Run linting
npm run lint

# Format code
npm run format
```

### Useful Commands

```bash
# Format Go code
go fmt ./...

# Run goimports
goimports -w .

# Run golangci-lint
golangci-lint run

# Generate code (embeds web UI assets)
go generate ./webui

# Build (SQLite included; no CGO, no build tags)
CGO_ENABLED=0 go build
```

## Port Forwarding

The dev container automatically forwards these ports:

- **8080**: Main application server
- **3000**: Web UI development server (when running `npm run dev`)

## Volumes

The dev container uses volumes for:

- **Go module cache**: Persisted across container rebuilds
- **Node modules**: Persisted across container rebuilds for faster npm installs
- **Source code**: Bind mounted for real-time editing

## Troubleshooting

### Container Build Issues

If the container fails to build:

1. Make sure Docker is running
2. Check your Docker Desktop/Engine memory allocation (recommend 4GB+)
3. Try rebuilding: `Dev Containers: Rebuild Container`

### SQLite Test Failures

If you see SQLite-related test failures, make sure you're using the SQLite
variants:

```bash
make test-sqlite        # Instead of make test
make test-race-sqlite   # Instead of make test-race
```

### Port Conflicts

If ports 8080 or 3000 are already in use:

1. Stop any local services using those ports
2. Or modify the `forwardPorts` setting in `devcontainer.json`

### Permission Issues

If you encounter permission issues:

```bash
# Fix ownership
sudo chown -R vscode:vscode /workspace

# Or rebuild the container
```

## Customization

You can customize the dev container by modifying:

- `.devcontainer/devcontainer.json`: VS Code settings, extensions, port
  forwarding
- `.devcontainer/Dockerfile`: Base image, installed packages, system
  configuration
- `.devcontainer/post-create.sh`: Setup commands run after container creation

## Performance Tips

- The first build may take several minutes as it downloads and installs all
  dependencies
- Subsequent builds are much faster due to Docker layer caching
- Go modules and npm packages are cached in volumes for faster rebuilds
- Use the SQLite test variants for better performance in the dev container
