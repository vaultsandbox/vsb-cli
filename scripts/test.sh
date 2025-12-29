#!/bin/bash
# Run all tests (unit + e2e) with combined coverage

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

# Default: run everything
SKIP_UNIT=false
SKIP_E2E=false
SKIP_COVERAGE=false
VERBOSE=false

for arg in "$@"; do
    case $arg in
        --skip-unit)
            SKIP_UNIT=true
            ;;
        --skip-e2e)
            SKIP_E2E=true
            ;;
        --skip-coverage)
            SKIP_COVERAGE=true
            ;;
        -v|--verbose)
            VERBOSE=true
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "By default, runs all tests (unit + e2e) with combined coverage."
            echo ""
            echo "Options:"
            echo "  --skip-unit      Skip unit tests"
            echo "  --skip-e2e       Skip e2e tests"
            echo "  --skip-coverage  Skip coverage collection"
            echo "  -v, --verbose    Verbose output"
            echo "  --help           Show this help"
            exit 0
            ;;
    esac
done

# Validate required env vars for e2e
if [ "$SKIP_E2E" = false ]; then
    if [ -z "$VAULTSANDBOX_API_KEY" ]; then
        echo "Error: VAULTSANDBOX_API_KEY not set"
        echo "Create a .env file with VAULTSANDBOX_API_KEY and VAULTSANDBOX_URL"
        echo "Or use --skip-e2e to run only unit tests"
        exit 1
    fi
    if [ -z "$VAULTSANDBOX_URL" ]; then
        echo "Error: VAULTSANDBOX_URL not set"
        echo "Or use --skip-e2e to run only unit tests"
        exit 1
    fi
    echo "Using API URL: $VAULTSANDBOX_URL"
fi

# Setup coverage directories
if [ "$SKIP_COVERAGE" = false ]; then
    rm -rf coverage
    mkdir -p coverage/unit coverage/e2e
fi

# Run unit tests
if [ "$SKIP_UNIT" = false ]; then
    echo ""
    echo "=== Running unit tests ==="

    UNIT_CMD="go test"
    if [ "$SKIP_COVERAGE" = false ]; then
        UNIT_CMD="$UNIT_CMD -coverprofile=coverage/unit.out"
    fi
    if [ "$VERBOSE" = true ]; then
        UNIT_CMD="$UNIT_CMD -v"
    fi
    UNIT_CMD="$UNIT_CMD ./internal/..."

    echo "Running: $UNIT_CMD"
    $UNIT_CMD
fi

# Run e2e tests
if [ "$SKIP_E2E" = false ]; then
    echo ""
    echo "=== Running e2e tests ==="

    # Build binary with coverage instrumentation
    if [ "$SKIP_COVERAGE" = false ]; then
        echo "Building vsb binary with coverage instrumentation..."
        go build -cover -o vsb ./cmd/vsb
        export GOCOVERDIR=coverage/e2e
    else
        echo "Building vsb binary..."
        go build -o vsb ./cmd/vsb
    fi

    E2E_CMD="go test -tags=e2e -timeout 10m"
    if [ "$VERBOSE" = true ]; then
        E2E_CMD="$E2E_CMD -v"
    fi
    E2E_CMD="$E2E_CMD ./e2e/..."

    echo "Running: $E2E_CMD"
    $E2E_CMD
fi

# Generate combined coverage report
if [ "$SKIP_COVERAGE" = false ]; then
    echo ""
    echo "=== Coverage Report ==="

    # Convert e2e binary coverage to text format (if e2e was run)
    if [ "$SKIP_E2E" = false ] && [ -d coverage/e2e ] && [ "$(ls -A coverage/e2e 2>/dev/null)" ]; then
        go tool covdata textfmt -i=coverage/e2e -o=coverage/e2e.out
    fi

    # Combine coverage files
    if [ "$SKIP_UNIT" = false ] && [ "$SKIP_E2E" = false ] && [ -f coverage/unit.out ] && [ -f coverage/e2e.out ]; then
        # Merge coverage files (remove mode line from second file)
        head -1 coverage/unit.out > coverage/combined.out
        tail -n +2 coverage/unit.out >> coverage/combined.out
        tail -n +2 coverage/e2e.out >> coverage/combined.out
        COVERAGE_FILE=coverage/combined.out
        echo "Combined unit + e2e coverage:"
    elif [ -f coverage/unit.out ]; then
        COVERAGE_FILE=coverage/unit.out
        echo "Unit test coverage:"
    elif [ -f coverage/e2e.out ]; then
        COVERAGE_FILE=coverage/e2e.out
        echo "E2E test coverage:"
    else
        echo "No coverage data collected"
        exit 0
    fi

    # Show summary
    go tool cover -func="$COVERAGE_FILE" | tail -1
    echo ""
    echo "To view HTML report: go tool cover -html=$COVERAGE_FILE"
fi
