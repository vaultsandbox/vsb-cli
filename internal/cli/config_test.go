package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"long key", "vsb_test1234567890abcdef", "vsb_tes...cdef"},
		{"exactly 11 chars", "12345678901", "1234567...8901"},
		{"short key", "short", "****"},
		{"empty key", "", "****"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := maskAPIKey(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConfigCmd_E2E(t *testing.T) {
	// Build the binary once for all tests
	binPath := filepath.Join(t.TempDir(), "vsb")
	cmd := exec.Command("go", "build", "-o", binPath, "../../cmd/vsb")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build binary: %s", output)

	runVSB := func(t *testing.T, configDir string, args ...string) (stdout, stderr string, exitCode int) {
		cmd := exec.Command(binPath, args...)
		cmd.Env = append(os.Environ(),
			"VSB_CONFIG_DIR="+configDir,
			"VSB_API_KEY=", // Clear any env var
		)

		var stdoutBuf, stderrBuf strings.Builder
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		err := cmd.Run()
		exitCode = 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		return stdoutBuf.String(), stderrBuf.String(), exitCode
	}

	t.Run("config set strategy sse", func(t *testing.T) {
		configDir := t.TempDir()

		stdout, stderr, code := runVSB(t, configDir, "config", "set", "strategy", "sse")
		assert.Equal(t, 0, code, "stderr: %s", stderr)
		assert.Contains(t, stdout, "Set strategy successfully")
	})

	t.Run("config set strategy polling", func(t *testing.T) {
		configDir := t.TempDir()

		stdout, stderr, code := runVSB(t, configDir, "config", "set", "strategy", "polling")
		assert.Equal(t, 0, code, "stderr: %s", stderr)
		assert.Contains(t, stdout, "Set strategy successfully")
	})

	t.Run("config set strategy invalid fails", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSB(t, configDir, "config", "set", "strategy", "invalid")
		assert.NotEqual(t, 0, code)
		assert.Contains(t, stderr, "invalid strategy")
	})

	t.Run("config show displays strategy", func(t *testing.T) {
		configDir := t.TempDir()

		// Set strategy first
		_, _, code := runVSB(t, configDir, "config", "set", "strategy", "polling")
		require.Equal(t, 0, code)

		// Show config
		stdout, stderr, code := runVSB(t, configDir, "config", "show")
		assert.Equal(t, 0, code, "stderr: %s", stderr)
		assert.Contains(t, stdout, "strategy:")
		assert.Contains(t, stdout, "polling")
	})

	t.Run("config show defaults strategy to sse", func(t *testing.T) {
		configDir := t.TempDir()

		stdout, stderr, code := runVSB(t, configDir, "config", "show")
		assert.Equal(t, 0, code, "stderr: %s", stderr)
		assert.Contains(t, stdout, "strategy:")
		assert.Contains(t, stdout, "sse")
	})

	t.Run("config show json includes strategy", func(t *testing.T) {
		configDir := t.TempDir()

		_, _, code := runVSB(t, configDir, "config", "set", "strategy", "polling")
		require.Equal(t, 0, code)

		stdout, stderr, code := runVSB(t, configDir, "config", "show", "--output", "json")
		assert.Equal(t, 0, code, "stderr: %s", stderr)
		assert.Contains(t, stdout, `"strategy"`)
		assert.Contains(t, stdout, `"polling"`)
	})

	t.Run("config set without value for non-strategy key fails", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSB(t, configDir, "config", "set", "api-key")
		assert.NotEqual(t, 0, code)
		assert.Contains(t, stderr, "value required")
	})

	t.Run("config set help shows strategy option", func(t *testing.T) {
		configDir := t.TempDir()

		stdout, _, code := runVSB(t, configDir, "config", "set", "--help")
		assert.Equal(t, 0, code)
		assert.Contains(t, stdout, "strategy")
	})

	t.Run("strategy persists across invocations", func(t *testing.T) {
		configDir := t.TempDir()

		// Set strategy
		_, _, code := runVSB(t, configDir, "config", "set", "strategy", "polling")
		require.Equal(t, 0, code)

		// Verify it persists
		stdout, _, code := runVSB(t, configDir, "config", "show")
		assert.Equal(t, 0, code)
		assert.Contains(t, stdout, "polling")

		// Change it
		_, _, code = runVSB(t, configDir, "config", "set", "strategy", "sse")
		require.Equal(t, 0, code)

		// Verify change persisted
		stdout, _, code = runVSB(t, configDir, "config", "show")
		assert.Equal(t, 0, code)
		assert.Contains(t, stdout, "sse")
	})
}
