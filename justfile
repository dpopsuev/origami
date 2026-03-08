# Origami — task runner
# Run `just` with no args to see available recipes.

set dotenv-load := false

bin_dir := "bin"
cmd     := "./cmd/origami"

# ─── Default ──────────────────────────────────────────────

# List available recipes
default:
    @just --list

# ─── Build ────────────────────────────────────────────────

# Build the origami CLI
build:
    @mkdir -p {{ bin_dir }}
    go build -o {{ bin_dir }}/origami {{ cmd }}

# Install origami to ~/.local/bin
install:
    go build -o ~/.local/bin/origami {{ cmd }}

# ─── Test ─────────────────────────────────────────────────

# Run all Go tests
test:
    go test ./...

# Run all Go tests with race detector
test-race:
    go test -race ./...

# Run all Go tests with verbose output
test-v:
    go test -v ./...

# ─── Lint ─────────────────────────────────────────────────

# Run go vet
vet:
    go vet ./...

# Run origami lint on all testdata YAMLs (strict profile)
lint-pipelines:
    @for f in testdata/*.yaml testdata/**/*.yaml; do echo "lint: $f"; origami lint --profile strict "$f"; done

# ─── Container Images ─────────────────────────────────────

# Build all OCI images
build-images: build-gateway build-rca build-knowledge build-llm-worker

# Build gateway image
build-gateway:
    docker build -t origami-gateway -f deploy/Dockerfile.gateway .

# Build RCA engine image
build-rca:
    docker build -t origami-rca -f deploy/Dockerfile.rca .

# Build knowledge engine image
build-knowledge:
    docker build -t origami-knowledge -f deploy/Dockerfile.knowledge .

# Build LLM worker image
build-llm-worker:
    docker build -t origami-llm-worker -f deploy/Dockerfile.llm-worker .

# ─── Clean ────────────────────────────────────────────────

# Remove build artifacts
clean:
    rm -rf {{ bin_dir }}
