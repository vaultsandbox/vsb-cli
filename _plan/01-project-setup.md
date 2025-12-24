# Phase 1.1: Project Setup

## Objective
Initialize the Go module structure, set up dependencies, and create the entry point.

## Tasks

### 1. Initialize Go Module

```bash
cd /home/vs/dev/vsb-cli
go mod init github.com/vaultsandbox/vsb-cli
```

### 2. Add Dependencies

```bash
# CLI framework
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest

# TUI framework
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest

# Local SDK (add replace directive)
```

### 3. Configure go.mod

```go
module github.com/vaultsandbox/vsb-cli

go 1.23

require (
    github.com/charmbracelet/bubbletea v1.2.4
    github.com/charmbracelet/bubbles v0.20.0
    github.com/charmbracelet/lipgloss v1.0.0
    github.com/spf13/cobra v1.8.1
    github.com/spf13/viper v1.19.0
    github.com/vaultsandbox/client-go v0.0.0
)

// Local SDK (not yet published)
replace github.com/vaultsandbox/client-go => /home/vs/Desktop/dev/client-go
```

### 4. Create Directory Structure

```bash
mkdir -p cmd/vsb
mkdir -p internal/cli
mkdir -p internal/config
mkdir -p internal/tui/watch
mkdir -p internal/tui/styles
mkdir -p internal/output
```

### 5. Create Entry Point

**File: `cmd/vsb/main.go`**

```go
package main

import (
    "os"

    "github.com/vaultsandbox/vsb-cli/internal/cli"
)

func main() {
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### 6. Create Root Command

**File: `internal/cli/root.go`**

```go
package cli

import (
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    cfgFile string
)

var rootCmd = &cobra.Command{
    Use:   "vsb",
    Short: "VaultSandbox CLI - Test email flows with quantum-safe encryption",
    Long: `vsb is a developer companion for testing email flows.

It provides temporary inboxes with end-to-end encryption using
quantum-safe algorithms (ML-KEM-768, ML-DSA-65).

The server never sees your email content - all decryption
happens locally on your machine.`,
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    cobra.OnInitialize(initConfig)

    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
        "config file (default is $HOME/.config/vsb/config.yaml)")

    // Global flags
    rootCmd.PersistentFlags().String("api-key", "", "API key (overrides config)")
    rootCmd.PersistentFlags().String("base-url", "", "API base URL")
    rootCmd.PersistentFlags().StringP("output", "o", "pretty",
        "Output format: pretty, json, minimal")

    // Bind flags to viper
    viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))
    viper.BindPFlag("base_url", rootCmd.PersistentFlags().Lookup("base-url"))
    viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        configDir, err := os.UserConfigDir()
        if err != nil {
            return
        }

        vsbConfigDir := filepath.Join(configDir, "vsb")
        viper.AddConfigPath(vsbConfigDir)
        viper.SetConfigName("config")
        viper.SetConfigType("yaml")
    }

    // Environment variables
    viper.SetEnvPrefix("VSB")
    viper.AutomaticEnv()

    // Read config (ignore error if not found)
    viper.ReadInConfig()
}
```

### 7. Create Output Styles

**File: `internal/output/printer.go`**

```go
package output

import (
    "github.com/charmbracelet/lipgloss"
)

var (
    // Colors
    Primary   = lipgloss.Color("#7C3AED")  // Purple
    Success   = lipgloss.Color("#10B981")  // Green
    Warning   = lipgloss.Color("#F59E0B")  // Amber
    Error     = lipgloss.Color("#EF4444")  // Red
    Muted     = lipgloss.Color("#6B7280")  // Gray

    // Styles
    TitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(Primary)

    SuccessStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(Success)

    ErrorStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(Error)

    MutedStyle = lipgloss.NewStyle().
        Foreground(Muted)

    // Box for important info
    BoxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(Primary).
        Padding(1, 2)

    // Email address highlight
    EmailStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#FFFFFF")).
        Background(Primary).
        Padding(0, 1)
)

// Success prints a success message with checkmark
func Success(msg string) string {
    return SuccessStyle.Render("✓ " + msg)
}

// Error prints an error message
func Error(msg string) string {
    return ErrorStyle.Render("✗ " + msg)
}

// Info prints an info message
func Info(msg string) string {
    return MutedStyle.Render("• " + msg)
}
```

## Verification

After completing these tasks, run:

```bash
cd /home/vs/dev/vsb-cli
go mod tidy
go build ./cmd/vsb
./vsb --help
```

Expected output should show the root command help text.

## Files Created

- `go.mod`
- `cmd/vsb/main.go`
- `internal/cli/root.go`
- `internal/output/printer.go`

## Next Steps

Proceed to [02-config-keystore.md](02-config-keystore.md) to implement configuration and keystore management.
