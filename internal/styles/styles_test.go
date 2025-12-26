package styles

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/client-go/authresults"
)

func TestScoreStyle(t *testing.T) {
	tests := []struct {
		name      string
		score     int
		wantStyle lipgloss.Style
	}{
		{"zero score", 0, FailStyle},
		{"below threshold", 59, FailStyle},
		{"at warn threshold", 60, WarnStyle},
		{"mid warn range", 70, WarnStyle},
		{"just below pass", 79, WarnStyle},
		{"at pass threshold", 80, PassStyle},
		{"perfect score", 100, PassStyle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreStyle(tt.score)
			// Compare by rendering a test string and checking they match
			assert.Equal(t, tt.wantStyle.Render("test"), got.Render("test"))
		})
	}
}

func TestFormatAuthResult(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"pass", "PASS"},
		{"PASS", "PASS"},
		{"Pass", "PASS"},
		{"fail", "FAIL"},
		{"FAIL", "FAIL"},
		{"hardfail", "FAIL"},
		{"softfail", "SOFTFAIL"},
		{"SOFTFAIL", "SOFTFAIL"},
		{"none", "NONE"},
		{"neutral", "NEUTRAL"},
		{"unknown", "unknown"}, // Unknown values returned as-is
		{"garbage", "garbage"}, // Unknown values returned as-is
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FormatAuthResult(tt.input)
			assert.Contains(t, got, tt.contains)
		})
	}
}

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name  string
		email *vaultsandbox.Email
		want  int
	}{
		{
			name:  "nil email auth results",
			email: &vaultsandbox.Email{AuthResults: nil},
			want:  50,
		},
		{
			name: "all pass",
			email: &vaultsandbox.Email{
				AuthResults: &authresults.AuthResults{
					SPF:        &authresults.SPFResult{Status: "pass"},
					DKIM:       []authresults.DKIMResult{{Status: "pass"}},
					DMARC:      &authresults.DMARCResult{Status: "pass"},
					ReverseDNS: &authresults.ReverseDNSResult{Verified: true},
				},
			},
			want: 100,
		},
		{
			name: "SPF only pass",
			email: &vaultsandbox.Email{
				AuthResults: &authresults.AuthResults{
					SPF: &authresults.SPFResult{Status: "pass"},
				},
			},
			want: 65,
		},
		{
			name: "DKIM only pass",
			email: &vaultsandbox.Email{
				AuthResults: &authresults.AuthResults{
					DKIM: []authresults.DKIMResult{{Status: "pass"}},
				},
			},
			want: 70,
		},
		{
			name: "DMARC only pass",
			email: &vaultsandbox.Email{
				AuthResults: &authresults.AuthResults{
					DMARC: &authresults.DMARCResult{Status: "pass"},
				},
			},
			want: 60,
		},
		{
			name: "ReverseDNS only pass",
			email: &vaultsandbox.Email{
				AuthResults: &authresults.AuthResults{
					ReverseDNS: &authresults.ReverseDNSResult{Verified: true},
				},
			},
			want: 55,
		},
		{
			name: "all fail",
			email: &vaultsandbox.Email{
				AuthResults: &authresults.AuthResults{
					SPF:        &authresults.SPFResult{Status: "fail"},
					DKIM:       []authresults.DKIMResult{{Status: "fail"}},
					DMARC:      &authresults.DMARCResult{Status: "fail"},
					ReverseDNS: &authresults.ReverseDNSResult{Verified: false},
				},
			},
			want: 50,
		},
		{
			name: "case insensitive pass",
			email: &vaultsandbox.Email{
				AuthResults: &authresults.AuthResults{
					SPF:   &authresults.SPFResult{Status: "PASS"},
					DKIM:  []authresults.DKIMResult{{Status: "Pass"}},
					DMARC: &authresults.DMARCResult{Status: "pAsS"},
				},
			},
			want: 95,
		},
		{
			name: "empty DKIM array",
			email: &vaultsandbox.Email{
				AuthResults: &authresults.AuthResults{
					DKIM: []authresults.DKIMResult{},
				},
			},
			want: 50,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateScore(tt.email)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRenderAuthResults(t *testing.T) {
	labelStyle := LabelStyle

	t.Run("nil auth returns warning", func(t *testing.T) {
		got := RenderAuthResults(nil, labelStyle, false)
		assert.Contains(t, got, "No authentication results available")
	})

	t.Run("compact mode with SPF", func(t *testing.T) {
		auth := &authresults.AuthResults{
			SPF: &authresults.SPFResult{
				Status: "pass",
				Domain: "example.com",
			},
		}
		got := RenderAuthResults(auth, labelStyle, false)
		assert.Contains(t, got, "SPF:")
		assert.Contains(t, got, "PASS")
		assert.Contains(t, got, "(example.com)")
	})

	t.Run("verbose mode with SPF", func(t *testing.T) {
		auth := &authresults.AuthResults{
			SPF: &authresults.SPFResult{
				Status: "pass",
				Domain: "example.com",
			},
		}
		got := RenderAuthResults(auth, labelStyle, true)
		assert.Contains(t, got, "SPF:")
		assert.Contains(t, got, "PASS")
		assert.Contains(t, got, "Domain:")
		assert.Contains(t, got, "example.com")
	})

	t.Run("compact mode with DKIM", func(t *testing.T) {
		auth := &authresults.AuthResults{
			DKIM: []authresults.DKIMResult{
				{Status: "pass", Domain: "example.com", Selector: "s1"},
			},
		}
		got := RenderAuthResults(auth, labelStyle, false)
		assert.Contains(t, got, "DKIM:")
		assert.Contains(t, got, "PASS")
		assert.Contains(t, got, "(example.com)")
	})

	t.Run("verbose mode with DKIM", func(t *testing.T) {
		auth := &authresults.AuthResults{
			DKIM: []authresults.DKIMResult{
				{Status: "pass", Domain: "example.com", Selector: "s1"},
			},
		}
		got := RenderAuthResults(auth, labelStyle, true)
		assert.Contains(t, got, "DKIM:")
		assert.Contains(t, got, "PASS")
		assert.Contains(t, got, "Selector:")
		assert.Contains(t, got, "s1")
		assert.Contains(t, got, "Domain:")
	})

	t.Run("compact mode with DMARC", func(t *testing.T) {
		auth := &authresults.AuthResults{
			DMARC: &authresults.DMARCResult{
				Status: "pass",
				Policy: "reject",
			},
		}
		got := RenderAuthResults(auth, labelStyle, false)
		assert.Contains(t, got, "DMARC:")
		assert.Contains(t, got, "PASS")
		assert.Contains(t, got, "(policy: reject)")
	})

	t.Run("verbose mode with DMARC", func(t *testing.T) {
		auth := &authresults.AuthResults{
			DMARC: &authresults.DMARCResult{
				Status: "pass",
				Policy: "reject",
			},
		}
		got := RenderAuthResults(auth, labelStyle, true)
		assert.Contains(t, got, "DMARC:")
		assert.Contains(t, got, "PASS")
		assert.Contains(t, got, "Policy:")
		assert.Contains(t, got, "reject")
	})

	t.Run("compact mode with ReverseDNS", func(t *testing.T) {
		auth := &authresults.AuthResults{
			ReverseDNS: &authresults.ReverseDNSResult{
				Verified: true,
				Hostname: "mail.example.com",
			},
		}
		got := RenderAuthResults(auth, labelStyle, false)
		assert.Contains(t, got, "Reverse DNS:")
		assert.Contains(t, got, "PASS")
		assert.Contains(t, got, "(mail.example.com)")
	})

	t.Run("verbose mode with ReverseDNS", func(t *testing.T) {
		auth := &authresults.AuthResults{
			ReverseDNS: &authresults.ReverseDNSResult{
				Verified: true,
				Hostname: "mail.example.com",
			},
		}
		got := RenderAuthResults(auth, labelStyle, true)
		assert.Contains(t, got, "Reverse DNS:")
		assert.Contains(t, got, "PASS")
		assert.Contains(t, got, "Hostname:")
		assert.Contains(t, got, "mail.example.com")
	})

	t.Run("all results compact", func(t *testing.T) {
		auth := &authresults.AuthResults{
			SPF:        &authresults.SPFResult{Status: "pass", Domain: "example.com"},
			DKIM:       []authresults.DKIMResult{{Status: "pass", Domain: "example.com"}},
			DMARC:      &authresults.DMARCResult{Status: "pass", Policy: "reject"},
			ReverseDNS: &authresults.ReverseDNSResult{Verified: true, Hostname: "mail.example.com"},
		}
		got := RenderAuthResults(auth, labelStyle, false)
		// Should have all four results
		lines := strings.Split(got, "\n")
		assert.GreaterOrEqual(t, len(lines), 4)
	})

	t.Run("missing optional fields", func(t *testing.T) {
		auth := &authresults.AuthResults{
			SPF: &authresults.SPFResult{Status: "pass"},
		}
		got := RenderAuthResults(auth, labelStyle, false)
		assert.Contains(t, got, "SPF:")
		assert.Contains(t, got, "PASS")
		assert.NotContains(t, got, "()")
	})
}
