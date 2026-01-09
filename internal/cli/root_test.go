package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_E2E(t *testing.T) {
	// Build the binary once for all tests
	binPath := filepath.Join(t.TempDir(), "vsb")
	cmd := exec.Command("go", "build", "-o", binPath, "../../cmd/vsb")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build binary: %s", output)

	t.Run("shows error when no keystore exists", func(t *testing.T) {
		tmpHome := t.TempDir()

		cmd := exec.Command(binPath)
		cmd.Env = append(os.Environ(),
			"HOME="+tmpHome,
			"XDG_CONFIG_HOME="+filepath.Join(tmpHome, ".config"),
		)

		output, err := cmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "no inboxes")
	})

	t.Run("shows error when keystore is empty", func(t *testing.T) {
		tmpHome := t.TempDir()
		configDir := filepath.Join(tmpHome, ".config", "vsb")
		require.NoError(t, os.MkdirAll(configDir, 0755))

		// Create empty keystore
		keystorePath := filepath.Join(configDir, "keystore.json")
		require.NoError(t, os.WriteFile(keystorePath, []byte(`{"inboxes":[]}`), 0600))

		cmd := exec.Command(binPath)
		cmd.Env = append(os.Environ(),
			"HOME="+tmpHome,
			"XDG_CONFIG_HOME="+filepath.Join(tmpHome, ".config"),
		)

		output, err := cmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "no inboxes found")
	})

	t.Run("shows version with --version flag", func(t *testing.T) {
		cmd := exec.Command(binPath, "--version")
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "vsb version")
	})

	t.Run("shows help with --help flag", func(t *testing.T) {
		cmd := exec.Command(binPath, "--help")
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "developer companion for testing email flows")
		assert.Contains(t, string(output), "inbox")
		assert.Contains(t, string(output), "email")
	})
}

func TestInitConfig(t *testing.T) {
	originalCfgFile := cfgFile
	t.Cleanup(func() {
		cfgFile = originalCfgFile
	})

	t.Run("uses custom config path when set", func(t *testing.T) {
		cfgFile = filepath.Join(t.TempDir(), "custom.yaml")
		assert.NotPanics(t, func() {
			initConfig()
		})
	})

	t.Run("uses default path when cfgFile empty", func(t *testing.T) {
		cfgFile = ""
		assert.NotPanics(t, func() {
			initConfig()
		})
	})
}

func TestRootCmdStructure(t *testing.T) {
	assert.Equal(t, "vsb", rootCmd.Use)
	assert.NotEmpty(t, rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)
	assert.Equal(t, Version, rootCmd.Version)

	// Check flags exist
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup("config"))
	assert.NotNil(t, rootCmd.PersistentFlags().Lookup("output"))

	// Check subcommands
	cmdNames := make([]string, 0)
	for _, cmd := range rootCmd.Commands() {
		cmdNames = append(cmdNames, cmd.Name())
	}
	assert.Contains(t, cmdNames, "inbox")
	assert.Contains(t, cmdNames, "email")
	assert.Contains(t, cmdNames, "export")
	assert.Contains(t, cmdNames, "import")
}
