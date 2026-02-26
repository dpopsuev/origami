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
    @just build
    @for f in testdata/*.yaml testdata/**/*.yaml; do echo "lint: $f"; {{ bin_dir }}/origami lint --profile strict "$f"; done

# ─── Clean ────────────────────────────────────────────────

# Remove build artifacts
clean:
    rm -rf {{ bin_dir }}
