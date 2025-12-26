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

With coverage:

```bash
go build -o vsb ./cmd/vsb
go test -tags=e2e -coverprofile=coverage.out -timeout 10m ./e2e/... && go tool cover -func=coverage.out
```

### All Tests with Coverage

```bash
go build -o vsb ./cmd/vsb
go test -tags=e2e -coverprofile=coverage.out -timeout 10m ./... && go tool cover -func=coverage.out
```

To view coverage in browser:

```bash
go tool cover -html=coverage.out
```