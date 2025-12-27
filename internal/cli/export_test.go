package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetExportPath(t *testing.T) {
	t.Run("uses email as default filename", func(t *testing.T) {
		result := getExportPath("", "test@example.com")
		assert.Equal(t, "test_example_com.json", result)
	})

	t.Run("sanitizes email for filename", func(t *testing.T) {
		result := getExportPath("", "user.name@domain.co.uk")
		assert.Equal(t, "user_name_domain_co_uk.json", result)
	})

	t.Run("respects --out flag", func(t *testing.T) {
		result := getExportPath("/custom/path/backup.json", "test@example.com")
		assert.Equal(t, "/custom/path/backup.json", result)
	})

	t.Run("respects relative --out flag", func(t *testing.T) {
		result := getExportPath("./backups/my-inbox.json", "test@example.com")
		assert.Equal(t, "./backups/my-inbox.json", result)
	})

	t.Run("--out flag takes precedence over email", func(t *testing.T) {
		// Even if email would generate a different name, --out wins
		result := getExportPath("custom.json", "different@email.com")
		assert.Equal(t, "custom.json", result)
	})
}
