package inbox

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInbox(t *testing.T) {
	t.Run("shows help when no arguments", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:   "test-inbox",
			Short: "Test command",
			Long:  "Test command for testing",
		}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := runInbox(cmd, []string{})

		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Test command for testing")
	})

	t.Run("returns error for unknown command", func(t *testing.T) {
		cmd := &cobra.Command{
			Use: "inbox",
		}

		err := runInbox(cmd, []string{"unknowncommand"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command")
		assert.Contains(t, err.Error(), "unknowncommand")
		assert.Contains(t, err.Error(), "inbox")
	})

	t.Run("returns error with full command path", func(t *testing.T) {
		parent := &cobra.Command{Use: "vsb"}
		child := &cobra.Command{Use: "inbox"}
		parent.AddCommand(child)

		err := runInbox(child, []string{"badcmd"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "vsb inbox")
	})
}
