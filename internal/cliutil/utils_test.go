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

func TestSubjectOrDefault(t *testing.T) {
	t.Run("empty string returns NoSubject", func(t *testing.T) {
		got := SubjectOrDefault("")
		assert.Equal(t, NoSubject, got)
	})

	t.Run("non-empty string returns as-is", func(t *testing.T) {
		got := SubjectOrDefault("Hello World")
		assert.Equal(t, "Hello World", got)
	})

	t.Run("whitespace-only returns as-is", func(t *testing.T) {
		got := SubjectOrDefault("   ")
		assert.Equal(t, "   ", got)
	})
}

func TestExtractTLSVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"TLSv1.3", "version=TLSv1.3", "TLSv1.3"},
		{"TLSv1.2", "version=TLSv1.2", "TLSv1.2"},
		{"TLSv1.1", "version=TLSv1.1", "TLSv1.1"},
		{"full header", "from mail.example.com (EHLO) with ESMTPS (version=TLSv1.3 cipher=TLS_AES_256_GCM_SHA384)", "TLSv1.3"},
		{"no match", "from mail.example.com with SMTP", ""},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTLSVersion(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractTLSCipher(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"AES 256", "cipher=TLS_AES_256_GCM_SHA384)", "TLS_AES_256_GCM_SHA384"},
		{"AES 128", "cipher=TLS_AES_128_GCM_SHA256)", "TLS_AES_128_GCM_SHA256"},
		{"full header", "from mail.example.com (EHLO) with ESMTPS (version=TLSv1.3 cipher=TLS_AES_256_GCM_SHA384)", "TLS_AES_256_GCM_SHA384"},
		{"no match", "from mail.example.com with SMTP", ""},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTLSCipher(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatExpiry(t *testing.T) {
	t.Run("expired returns expired", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		got := FormatExpiry(past)
		assert.Equal(t, "expired", got)
	})

	t.Run("future returns duration", func(t *testing.T) {
		future := time.Now().Add(30 * time.Minute)
		got := FormatExpiry(future)
		assert.Equal(t, "30m", got)
	})

	t.Run("far future returns days", func(t *testing.T) {
		future := time.Now().Add(48 * time.Hour)
		got := FormatExpiry(future)
		assert.Equal(t, "2d", got)
	})
}

func TestIsExpired(t *testing.T) {
	t.Run("past time is expired", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		assert.True(t, IsExpired(past))
	})

	t.Run("future time is not expired", func(t *testing.T) {
		future := time.Now().Add(1 * time.Hour)
		assert.False(t, IsExpired(future))
	})
}
