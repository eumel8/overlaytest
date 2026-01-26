# Overlay Network Test

Test Overlay Network of your Kubernetes Cluster

This is a [Go client](https://github.com/kubernetes/client-go) of the [Overlay Network Test](https://github.com/mcsps/use-cases/blob/master/README.md#k8s-overlay-network-test), a shell script paired with a DaemonSet to check connectivity in Overlay Network in Kubernetes Cluster.

## Requirements

* kube-config with connection to a working Kubernetes Cluster
* access `kube-system` namespace where the application will deploy (needs privileged mode for network ping command)

## Usage

Download artifact from [Release Page](https://github.com/eumel8/overlaytest/releases) and execute:

```bash
# Basic usage
./overlaytest

# Show version
./overlaytest -version

# Reuse existing deployment
./overlaytest -reuse

# Custom kubeconfig
./overlaytest -kubeconfig /path/to/kubeconfig
```

### Version Configuration

The application version can be configured in multiple ways (priority order):

1. **Environment variable** (runtime):
   ```bash
   APP_VERSION=1.0.7 ./overlaytest -version
   ```

2. **Build-time ldflags** (compile time):
   ```bash
   go build -ldflags "-X github.com/eumel8/overlaytest/pkg/overlaytest.Version=1.0.7" -o overlaytest ./cmd/overlaytest
   ```

3. **Default version**: Falls back to hardcoded version (1.0.6)

## Project Structure

The project follows standard Go layout:

```
overlaytest/
├── cmd/overlaytest/          # Main application entry point
│   └── main.go
├── pkg/overlaytest/          # Library code
│   ├── version.go           # Version management
│   ├── config.go            # Configuration handling
│   ├── client.go            # Kubernetes client setup
│   ├── daemonset.go         # DaemonSet management
│   ├── network.go           # Network testing logic
│   └── *_test.go            # Unit tests
├── Dockerfile               # Container image definition
└── .github/workflows/       # CI/CD pipelines
```

## Container Image

**Current Image**: `ghcr.io/eumel8/overlaytest:latest`

The project includes a minimal Alpine-based container image (~10MB compressed) with:
- bash shell
- ping command
- Non-root user (UID 1000)

### Building the Container Image Locally

```bash
docker build -t overlaytest:local .
docker run --rm overlaytest:local ping -c 2 8.8.8.8
```

### Using a Custom Image

You can specify a custom container image:

```go
config := overlaytest.DefaultConfig()
config.Image = "your-registry/your-image:tag"
```

**Previous Image** (deprecated): `mtr.devops.telekom.de/mcsps/swiss-army-knife:latest`
- [source](https://github.com/mcsps/swiss-army-knife/tree/mcsps)
- [repo](https://mtr.devops.telekom.de/repository/mcsps/swiss-army-knife?tab=tags)

## Security & Compliance

The DaemonSet includes comprehensive security context configuration:

### Container Security Context
- ✅ **Privileged mode**: Required for ping operations
- ✅ **Non-root user**: Runs as UID/GID 1000
- ✅ **Read-only root filesystem**: Enhanced security
- ✅ **Seccomp profile**: RuntimeDefault (Kubernetes security standard)
- ✅ **Resource limits**: CPU (100m-200m), Memory (64Mi-128Mi)

### Compliance
- ✅ **Kyverno policies**: Meets resource and seccomp requirements
- ✅ **Pod Security Standards**: Compatible with restricted policies (except privileged requirement)
- ✅ **Issue #179**: All requirements addressed

## Building from Source

### Prerequisites
- Go 1.25 or later
- Docker (for container image builds)

### Build the Binary

```bash
# Clone the repository
git clone https://github.com/eumel8/overlaytest.git
cd overlaytest

# Build with default version
go build -o overlaytest ./cmd/overlaytest

# Build with custom version
go build -ldflags "-X github.com/eumel8/overlaytest/pkg/overlaytest.Version=1.0.7" \
  -o overlaytest ./cmd/overlaytest

# Run tests
go test -v ./...
```

### Build Multi-Arch Container Image

```bash
# Build for multiple architectures
docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/eumel8/overlaytest:latest .

# Build and push (requires authentication)
docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/eumel8/overlaytest:latest --push .
```

## CI/CD Pipelines

The project includes automated GitHub Actions workflows:

- **`release.yaml`**: Builds multi-platform binaries on release
- **`docker-build.yaml`**: Builds and pushes container images
- **`e2e-test.yaml`**: End-to-end tests with Kind clusters
- **`coverage.yaml`**: Code coverage reporting

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test -v ./pkg/overlaytest/
```

### Project Layout

- `cmd/overlaytest/`: Main application entry point
- `pkg/overlaytest/`: Reusable library code
- `Dockerfile`: Container image definition
- `.github/workflows/`: CI/CD automation

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `go test ./...` to verify
5. Submit a pull request

## Credits

Frank Kloeker f.kloeker@telekom.de

Life is for sharing. If you have an issue with the code or want to improve it, feel free to open an issue or an pull request.
