version := `git describe --tags --always --dirty 2>/dev/null || echo "0.0.1-dev"`
commit := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
build_date := `date -u +%Y-%m-%dT%H:%M:%SZ`

ldflags := "-X gitlab.com/dcresp0/rudder/pkg/rudder.Version=" + version + \
           " -X gitlab.com/dcresp0/rudder/pkg/rudder.Commit=" + commit + \
           " -X gitlab.com/dcresp0/rudder/pkg/rudder.BuildDate=" + build_date

# Default recipe — list all available recipes
default:
    @just --list

# Build the rudder binary
build:
    go build -ldflags="{{ldflags}}" -o bin/rudder ./cmd/rudder

# Install to $GOPATH/bin
install:
    go install -ldflags="{{ldflags}}" ./cmd/rudder

# Run all tests
test:
    go test ./...

# Run tests with coverage report
test-coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out

# Lint with golangci-lint
lint:
    golangci-lint run ./...

# Format code (gofmt + goimports)
fmt:
    gofmt -w .
    goimports -w .

# GoReleaser snapshot build (no publish)
release-dry:
    goreleaser release --snapshot --clean

# Remove build artifacts
clean:
    rm -rf bin/ dist/ coverage.out completions/

# Generate shell completions
completions:
    mkdir -p completions
    go run ./cmd/rudder completion bash > completions/rudder.bash
    go run ./cmd/rudder completion zsh > completions/rudder.zsh
    go run ./cmd/rudder completion fish > completions/rudder.fish

# Run rudder locally (pass args: `just run use`, `just run envs`)
run *args:
    go run ./cmd/rudder {{args}}

# Tidy go modules
tidy:
    go mod tidy

# Run fmt, lint, and test (pre-commit / CI equivalent)
check: fmt lint test
