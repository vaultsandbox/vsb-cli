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

E2E tests require a running VaultSandbox server and SMTP access. Create a `.env` file:

```bash
VAULTSANDBOX_URL=https://your-gateway.vsx.email
VAULTSANDBOX_API_KEY=your-api-key
SMTP_HOST=your-gateway.vsx.email
SMTP_PORT=25
```

Run tests:

```bash
./scripts/test.sh              # Run e2e tests
./scripts/test.sh --coverage   # Run with coverage
./scripts/test.sh --coverage -v # Verbose output
```

To view coverage in browser:

```bash
go tool cover -html=coverage.out
```