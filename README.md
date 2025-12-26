# vsb-cli

Command-line interface for VaultSandbox.

## Build

```bash
go build -o vsb ./cmd/vsb
```

## Run

```bash
./vsb
```

## Tests

### Unit Tests

```bash
go test ./...
```

With coverage:

```bash
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
```

### E2E Tests

E2E tests require a running VaultSandbox server and SMTP access. Set the following environment variables (or use a `.env` file):

```bash
VAULTSANDBOX_URL=https://your-gateway.vsx.email
VAULTSANDBOX_API_KEY=your-api-key
SMTP_HOST=your-gateway.vsx.email
SMTP_PORT=25
```

Run E2E tests:

```bash
go build -o vsb ./cmd/vsb && go test -tags=e2e -v -timeout 10m ./e2e/...
```

With coverage (requires Go 1.20+):

```bash
# Build binary with coverage instrumentation
go build -cover -o vsb ./cmd/vsb

# Run e2e tests with coverage directory
rm -rf coverage && mkdir -p coverage
GOCOVERDIR=coverage go test -tags=e2e -v -timeout 10m ./e2e/...

# Convert and display coverage
go tool covdata textfmt -i=coverage -o=coverage.out
go tool cover -func=coverage.out
```

### All Tests with Coverage

```bash
# Unit tests coverage
go test -coverprofile=coverage-unit.out ./...

# E2E tests coverage (binary instrumentation)
go build -cover -o vsb ./cmd/vsb
rm -rf coverage && mkdir -p coverage
GOCOVERDIR=coverage go test -tags=e2e -timeout 10m ./e2e/...
go tool covdata textfmt -i=coverage -o=coverage-e2e.out

# View coverage
go tool cover -func=coverage-unit.out
go tool cover -func=coverage-e2e.out
```

To view coverage in browser:

```bash
go tool cover -html=coverage-unit.out
go tool cover -html=coverage-e2e.out
```