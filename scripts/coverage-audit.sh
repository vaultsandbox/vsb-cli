#!/bin/bash
# Coverage Audit Script for vsb-cli
# Generates detailed coverage reports for all internal packages.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== VSB-CLI Coverage Audit ==="
echo ""

# Change to project root
cd "$(dirname "$0")/.."

# Clean up previous coverage files
rm -f coverage.out coverage.html

# Generate coverage for all internal packages
echo "Generating coverage for internal packages..."
go test -coverprofile=coverage.out ./internal/...

if [ ! -f coverage.out ]; then
    echo -e "${RED}Error: Failed to generate coverage.out${NC}"
    exit 1
fi

# Show coverage by package
echo ""
echo "=== Coverage by Package ==="
go tool cover -func=coverage.out | grep -E "^github.com/vaultsandbox/vsb-cli/internal/" | \
    awk -F'/' '{
        # Extract package name (e.g., internal/browser)
        pkg = $(NF-1)"/"$NF
        gsub(/:.*/, "", pkg)
        print pkg
    }' | sort -u | while read pkg; do
    coverage=$(go tool cover -func=coverage.out | grep "$pkg" | tail -1 | awk '{print $NF}')
    echo "$pkg: $coverage"
done

# Show uncovered functions (0.0%)
echo ""
echo "=== Uncovered Functions (0.0%) ==="
uncovered=$(go tool cover -func=coverage.out | grep "0.0%" | head -20)
if [ -z "$uncovered" ]; then
    echo -e "${GREEN}No uncovered functions found!${NC}"
else
    echo "$uncovered"
    uncovered_count=$(go tool cover -func=coverage.out | grep -c "0.0%" || true)
    if [ "$uncovered_count" -gt 20 ]; then
        echo "... and $((uncovered_count - 20)) more uncovered functions"
    fi
fi

# Show total coverage
echo ""
echo "=== Total Coverage ==="
total=$(go tool cover -func=coverage.out | tail -1)
coverage_pct=$(echo "$total" | awk '{print $NF}' | sed 's/%//')

# Color-code the output based on coverage percentage
if (( $(echo "$coverage_pct >= 70" | bc -l) )); then
    echo -e "${GREEN}$total${NC}"
elif (( $(echo "$coverage_pct >= 50" | bc -l) )); then
    echo -e "${YELLOW}$total${NC}"
else
    echo -e "${RED}$total${NC}"
fi

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
echo ""
echo "HTML report generated: coverage.html"
echo ""

# Coverage targets comparison
echo "=== Coverage Targets ==="
echo "Package              | Target | Status"
echo "---------------------|--------|-------"

check_coverage() {
    local pkg=$1
    local target=$2

    # Get coverage for this specific package from the test output
    local actual=$(go test -cover "./$pkg/..." 2>/dev/null | grep -oP 'coverage: \K[0-9.]+' | head -1)

    if [ -z "$actual" ]; then
        actual="0.0"
    fi

    if (( $(echo "$actual >= $target" | bc -l) )); then
        status="${GREEN}PASS${NC}"
    else
        status="${RED}FAIL${NC}"
    fi

    printf "%-20s | %5.1f%% | $status (%.1f%%)\n" "$pkg" "$target" "$actual"
}

# These values should match the targets in the phase-5 documentation
check_coverage "internal/styles" 90.0
check_coverage "internal/files" 90.0
check_coverage "internal/config" 85.0
check_coverage "internal/cli" 70.0
check_coverage "internal/tui" 60.0
check_coverage "internal/browser" 50.0

echo ""
echo "=== Summary ==="
echo "To view detailed coverage, open coverage.html in a browser."
echo "To run specific package tests with coverage:"
echo "  go test -coverprofile=pkg.out -covermode=atomic ./internal/browser/..."
echo "  go tool cover -html=pkg.out"
