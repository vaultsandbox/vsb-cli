package cliutil

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

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
			got := FormatDuration(tt.input)
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
			got := FormatRelativeTime(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("old date shows month day", func(t *testing.T) {
		old := now.Add(-30 * 24 * time.Hour)
		got := FormatRelativeTime(old)
		// Should be in "Jan 2" format
		assert.Contains(t, got, old.Format("Jan"))
	})
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
			got := SanitizeFilename(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetOutput(t *testing.T) {
	t.Run("returns flag value when set", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("output", "pretty", "output format")
		cmd.Flags().Set("output", "json")

		got := GetOutput(cmd)
		assert.Equal(t, "json", got)
	})

	t.Run("returns default when flag not set", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("output", "pretty", "output format")
		// Don't set the flag

		got := GetOutput(cmd)
		// Should return the config default (pretty)
		assert.NotEmpty(t, got)
	})

	t.Run("returns default when no output flag", func(t *testing.T) {
		cmd := &cobra.Command{}
		// No output flag defined

		got := GetOutput(cmd)
		assert.NotEmpty(t, got)
	})
}

func TestOutputJSON(t *testing.T) {
	t.Run("outputs valid JSON", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		err := OutputJSON(data)
		assert.NoError(t, err)
	})

	t.Run("handles nested structures", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "test",
			"nested": map[string]int{
				"count": 42,
			},
		}
		err := OutputJSON(data)
		assert.NoError(t, err)
	})

	t.Run("handles arrays", func(t *testing.T) {
		data := []string{"a", "b", "c"}
		err := OutputJSON(data)
		assert.NoError(t, err)
	})
}
