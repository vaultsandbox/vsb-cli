package cli

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestParseTTL(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		// Valid inputs
		{"1h", time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"1d", 24 * time.Hour, false},
		{"90s", 90 * time.Second, false},
		{"2h30m", 2*time.Hour + 30*time.Minute, false},

		// Invalid inputs
		{"", 0, true},
		{"x", 0, true},
		{"1y", 0, true},      // Year not supported
		{"-1h", -time.Hour, false}, // time.ParseDuration allows negative
		{"h", 0, true},       // No number
		{"d", 0, true},       // No number for days
		{"abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseTTL(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{30 * time.Minute, "30m"},
		{59 * time.Minute, "59m"},
		{60 * time.Minute, "1h"},
		{90 * time.Minute, "1h"},
		{23 * time.Hour, "23h"},
		{24 * time.Hour, "1d"},
		{48 * time.Hour, "2d"},
		{7 * 24 * time.Hour, "7d"},
		{0, "0m"},
		{1 * time.Minute, "1m"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatDuration(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		input time.Time
		want  string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"minutes ago", now.Add(-5 * time.Minute), "5m ago"},
		{"hours ago", now.Add(-3 * time.Hour), "3h ago"},
		{"days ago", now.Add(-2 * 24 * time.Hour), "2d ago"},
		{"week ago", now.Add(-6 * 24 * time.Hour), "6d ago"},
		// After 7 days, falls back to date format
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRelativeTime(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("old date shows month day", func(t *testing.T) {
		old := now.Add(-30 * 24 * time.Hour)
		got := formatRelativeTime(old)
		// Should be in "Jan 2" format
		assert.Contains(t, got, old.Format("Jan"))
	})
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is too long", 10, "this is t…"},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "ab…"},
		{"hello world", 5, "hell…"},
		{"a", 1, "a"},
		{"ab", 1, "…"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncate(tt.input, tt.max)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"test@example.com", "test_example_com"},
		{"user.name@domain.co.uk", "user_name_domain_co_uk"},
		{"simple", "simple"},
		{"with-dash", "with-dash"},
		{"with_underscore", "with_underscore"},
		{"UPPERCASE@DOMAIN.COM", "UPPERCASE_DOMAIN_COM"},
		{"123@456.789", "123_456_789"},
		{"special!#$%chars", "specialchars"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"sk-ant-1234567890abcdef", "sk-ant-...cdef"},
		{"short", "****"},
		{"12345678901", "1234567...8901"}, // Exactly 11 chars
		{"", "****"},
		{"1234567890", "****"}, // 10 chars, below threshold
		{"abcdefghijk", "abcdefg...hijk"}, // 11 chars
		{"a", "****"},
		{"12345", "****"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := maskAPIKey(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetOutput(t *testing.T) {
	t.Run("returns flag value when set", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("output", "pretty", "output format")
		cmd.Flags().Set("output", "json")

		got := getOutput(cmd)
		assert.Equal(t, "json", got)
	})

	t.Run("returns default when flag not set", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("output", "pretty", "output format")
		// Don't set the flag

		got := getOutput(cmd)
		// Should return the config default (pretty)
		assert.NotEmpty(t, got)
	})

	t.Run("returns default when no output flag", func(t *testing.T) {
		cmd := &cobra.Command{}
		// No output flag defined

		got := getOutput(cmd)
		assert.NotEmpty(t, got)
	})
}

func TestOutputJSON(t *testing.T) {
	t.Run("outputs valid JSON", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		err := outputJSON(data)
		assert.NoError(t, err)
	})

	t.Run("handles nested structures", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "test",
			"nested": map[string]int{
				"count": 42,
			},
		}
		err := outputJSON(data)
		assert.NoError(t, err)
	})

	t.Run("handles arrays", func(t *testing.T) {
		data := []string{"a", "b", "c"}
		err := outputJSON(data)
		assert.NoError(t, err)
	})
}
