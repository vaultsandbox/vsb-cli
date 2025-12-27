package email

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildWaitOptions(t *testing.T) {
	// Helper to reset globals after test
	resetWaitFlags := func() {
		waitForSubject = ""
		waitForSubjectRegex = ""
		waitForFrom = ""
		waitForFromRegex = ""
	}

	t.Run("no filters returns only timeout option", func(t *testing.T) {
		resetWaitFlags()

		opts, err := buildWaitOptions(30 * time.Second)
		require.NoError(t, err)
		// Should have at least the timeout option
		assert.NotEmpty(t, opts)
	})

	t.Run("subject filter adds option", func(t *testing.T) {
		resetWaitFlags()
		waitForSubject = "Test Subject"

		opts, err := buildWaitOptions(30 * time.Second)
		require.NoError(t, err)
		// Should have timeout + subject options
		assert.Len(t, opts, 2)

		resetWaitFlags()
	})

	t.Run("subject regex filter adds option", func(t *testing.T) {
		resetWaitFlags()
		waitForSubjectRegex = "^Test.*"

		opts, err := buildWaitOptions(30 * time.Second)
		require.NoError(t, err)
		assert.Len(t, opts, 2)

		resetWaitFlags()
	})

	t.Run("invalid subject regex returns error", func(t *testing.T) {
		resetWaitFlags()
		waitForSubjectRegex = "[invalid"

		_, err := buildWaitOptions(30 * time.Second)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid subject regex")

		resetWaitFlags()
	})

	t.Run("from filter adds option", func(t *testing.T) {
		resetWaitFlags()
		waitForFrom = "sender@example.com"

		opts, err := buildWaitOptions(30 * time.Second)
		require.NoError(t, err)
		assert.Len(t, opts, 2)

		resetWaitFlags()
	})

	t.Run("from regex filter adds option", func(t *testing.T) {
		resetWaitFlags()
		waitForFromRegex = "@example\\.com$"

		opts, err := buildWaitOptions(30 * time.Second)
		require.NoError(t, err)
		assert.Len(t, opts, 2)

		resetWaitFlags()
	})

	t.Run("invalid from regex returns error", func(t *testing.T) {
		resetWaitFlags()
		waitForFromRegex = "[invalid"

		_, err := buildWaitOptions(30 * time.Second)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid from regex")

		resetWaitFlags()
	})

	t.Run("combined filters (AND logic)", func(t *testing.T) {
		resetWaitFlags()
		waitForSubject = "Welcome"
		waitForFrom = "noreply@example.com"

		opts, err := buildWaitOptions(30 * time.Second)
		require.NoError(t, err)
		// timeout + subject + from = 3 options
		assert.Len(t, opts, 3)

		resetWaitFlags()
	})

	t.Run("all filters together", func(t *testing.T) {
		resetWaitFlags()
		waitForSubject = "Subject"
		waitForSubjectRegex = "Sub.*"
		waitForFrom = "from@test.com"
		waitForFromRegex = "@test\\.com$"

		opts, err := buildWaitOptions(60 * time.Second)
		require.NoError(t, err)
		// timeout + 4 filters = 5 options
		assert.Len(t, opts, 5)

		resetWaitFlags()
	})

	t.Run("various timeout durations", func(t *testing.T) {
		resetWaitFlags()

		testCases := []time.Duration{
			1 * time.Second,
			30 * time.Second,
			1 * time.Minute,
			5 * time.Minute,
			1 * time.Hour,
		}

		for _, timeout := range testCases {
			opts, err := buildWaitOptions(timeout)
			require.NoError(t, err)
			assert.NotEmpty(t, opts)
		}
	})

	t.Run("regex patterns work correctly", func(t *testing.T) {
		resetWaitFlags()

		// Test various valid regex patterns
		validPatterns := []string{
			"^start",
			"end$",
			".*",
			"[a-z]+",
			"\\d{4}",
			"hello|world",
			"(?i)case-insensitive",
		}

		for _, pattern := range validPatterns {
			waitForSubjectRegex = pattern
			opts, err := buildWaitOptions(30 * time.Second)
			require.NoError(t, err, "pattern %q should be valid", pattern)
			assert.NotEmpty(t, opts)
		}

		resetWaitFlags()
	})
}
