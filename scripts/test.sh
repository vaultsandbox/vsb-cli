#!/bin/bash
# Run e2e tests with coverage

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Load .env if it exists
if [ -f .env ]; then
    echo "Loading .env file..."
    set -a
    source .env
    set +a
fi

# Parse arguments
COVERAGE=false
VERBOSE=false

for arg in "$@"; do
    case $arg in
        --coverage)
            COVERAGE=true
            ;;
        -v|--verbose)
            VERBOSE=true
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --coverage     Generate coverage report"
            echo "  -v, --verbose  Verbose output"
            echo "  --help         Show this help"
            exit 0
            ;;
    esac
done

# Validate required env vars
if [ -z "$VAULTSANDBOX_API_KEY" ]; then
    echo "Error: VAULTSANDBOX_API_KEY not set"
    echo "Create a .env file with VAULTSANDBOX_API_KEY and VAULTSANDBOX_URL"
    exit 1
fi
if [ -z "$VAULTSANDBOX_URL" ]; then
    echo "Error: VAULTSANDBOX_URL not set"
    exit 1
fi

echo "Using API URL: $VAULTSANDBOX_URL"

# Build binary (with coverage instrumentation if needed)
if [ "$COVERAGE" = true ]; then
    echo "Building vsb binary with coverage instrumentation..."
    go build -cover -o vsb ./cmd/vsb
    rm -rf coverage
    mkdir -p coverage
    export GOCOVERDIR=coverage
else
    echo "Building vsb binary..."
    go build -o vsb ./cmd/vsb
fi

# Build test command
CMD="go test -tags=e2e -timeout 10m"

if [ "$VERBOSE" = true ]; then
    CMD="$CMD -v"
fi

CMD="$CMD ./e2e/..."

echo "Running: $CMD"
$CMD

if [ "$COVERAGE" = true ]; then
    echo ""
    echo "Processing coverage data..."
    go tool covdata textfmt -i=coverage -o=coverage.out
    echo ""
    echo "Coverage summary:"
    go tool cover -func=coverage.out | tail -1
    echo ""
    echo "To view HTML report: go tool cover -html=coverage.out"
fi
