package email

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vaultsandbox "github.com/vaultsandbox/client-go"
)

// mockEmailFetcher creates a mock implementation for testing
func mockEmailFetcher(email *vaultsandbox.Email, err error) EmailFetcher {
	return func(ctx context.Context, emailID, emailFlag string) (*vaultsandbox.Email, *vaultsandbox.Inbox, func(), error) {
		return email, nil, func() {}, err
	}
}

// createTestCommand creates a test cobra command with the output flag
func createTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "test",
		RunE: runURL,
	}
	cmd.Flags().StringP("output", "o", "", "Output format")
	return cmd
}

// resetURLTestState resets global state after each test
func resetURLTestState(oldFetcher EmailFetcher, oldOpenURL func(string) error, oldURLOpen int) {
	getEmailByIDOrLatestFunc = oldFetcher
	openURLInBrowserFunc = oldOpenURL
	urlOpen = oldURLOpen
}

// captureURLStdout captures stdout during function execution
func captureURLStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestRunURLListURLs(t *testing.T) {
	t.Run("lists URLs in text format", func(t *testing.T) {
		// Save and restore state
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{
				"https://example.com/verify",
				"https://example.com/dashboard",
				"https://example.com/unsubscribe",
			},
		}, nil)

		cmd := createTestCommand()
		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Contains(t, output, "1. https://example.com/verify")
		assert.Contains(t, output, "2. https://example.com/dashboard")
		assert.Contains(t, output, "3. https://example.com/unsubscribe")
	})

	t.Run("lists URLs in JSON format", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{
				"https://example.com/link1",
				"https://example.com/link2",
			},
		}, nil)

		cmd := createTestCommand()
		cmd.Flags().Set("output", "json")

		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Contains(t, output, "https://example.com/link1")
		assert.Contains(t, output, "https://example.com/link2")
		assert.Contains(t, output, "[")
		assert.Contains(t, output, "]")
	})

	t.Run("handles single URL", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{"https://only.url/here"},
		}, nil)

		cmd := createTestCommand()
		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Contains(t, output, "1. https://only.url/here")
	})
}

func TestRunURLNoURLs(t *testing.T) {
	t.Run("prints message when no URLs in text format", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{},
		}, nil)

		cmd := createTestCommand()
		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Contains(t, output, "No URLs found in email")
	})

	t.Run("returns empty array in JSON format when no URLs", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{},
		}, nil)

		cmd := createTestCommand()
		cmd.Flags().Set("output", "json")

		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Contains(t, output, "[]")
	})

	t.Run("handles nil Links slice", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: nil,
		}, nil)

		cmd := createTestCommand()
		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Contains(t, output, "No URLs found in email")
	})
}

func TestRunURLOpenURL(t *testing.T) {
	t.Run("opens first URL with --open 1", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 1
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{
				"https://example.com/first",
				"https://example.com/second",
			},
		}, nil)

		var openedURL string
		openURLInBrowserFunc = func(url string) error {
			openedURL = url
			return nil
		}

		cmd := createTestCommand()
		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Equal(t, "https://example.com/first", openedURL)
		assert.Contains(t, output, "Opening: https://example.com/first")
	})

	t.Run("opens second URL with --open 2", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 2
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{
				"https://example.com/first",
				"https://example.com/second",
				"https://example.com/third",
			},
		}, nil)

		var openedURL string
		openURLInBrowserFunc = func(url string) error {
			openedURL = url
			return nil
		}

		cmd := createTestCommand()
		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Equal(t, "https://example.com/second", openedURL)
		assert.Contains(t, output, "Opening: https://example.com/second")
	})

	t.Run("opens last URL with matching index", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 3
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{
				"https://example.com/first",
				"https://example.com/second",
				"https://example.com/third",
			},
		}, nil)

		var openedURL string
		openURLInBrowserFunc = func(url string) error {
			openedURL = url
			return nil
		}

		cmd := createTestCommand()
		captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Equal(t, "https://example.com/third", openedURL)
	})
}

func TestRunURLOpenURLOutOfRange(t *testing.T) {
	t.Run("returns error when index exceeds URL count", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 5
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{
				"https://example.com/only",
			},
		}, nil)

		cmd := createTestCommand()
		err := runURL(cmd, []string{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "URL index 5 out of range (1-1)")
	})

	t.Run("returns error when index is 1 but no URLs", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		// With urlOpen > 0 but no URLs, we should get "No URLs found" message first
		// But if there are 0 URLs, the check happens before the urlOpen check
		urlOpen = 1
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{},
		}, nil)

		cmd := createTestCommand()
		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			// No error because "no URLs" check happens first
			require.NoError(t, err)
		})

		assert.Contains(t, output, "No URLs found in email")
	})

	t.Run("returns detailed error for index 10 with 3 URLs", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 10
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{
				"https://example.com/one",
				"https://example.com/two",
				"https://example.com/three",
			},
		}, nil)

		cmd := createTestCommand()
		err := runURL(cmd, []string{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "URL index 10 out of range (1-3)")
	})
}

func TestRunURLErrorHandling(t *testing.T) {
	t.Run("returns error when email fetch fails", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		getEmailByIDOrLatestFunc = mockEmailFetcher(nil, errors.New("failed to fetch email"))

		cmd := createTestCommand()
		err := runURL(cmd, []string{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch email")
	})

	t.Run("returns error when browser open fails", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 1
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{"https://example.com/url"},
		}, nil)

		openURLInBrowserFunc = func(url string) error {
			return errors.New("browser not found")
		}

		cmd := createTestCommand()
		captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "browser not found")
		})
	})
}

func TestRunURLWithEmailID(t *testing.T) {
	t.Run("passes email ID to fetcher", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		var receivedEmailID string
		getEmailByIDOrLatestFunc = func(ctx context.Context, emailID, emailFlag string) (*vaultsandbox.Email, *vaultsandbox.Inbox, func(), error) {
			receivedEmailID = emailID
			return &vaultsandbox.Email{
				Links: []string{"https://example.com"},
			}, nil, func() {}, nil
		}

		cmd := createTestCommand()
		captureURLStdout(t, func() {
			err := runURL(cmd, []string{"test-email-123"})
			require.NoError(t, err)
		})

		assert.Equal(t, "test-email-123", receivedEmailID)
	})

	t.Run("uses empty string when no email ID provided", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		var receivedEmailID string
		getEmailByIDOrLatestFunc = func(ctx context.Context, emailID, emailFlag string) (*vaultsandbox.Email, *vaultsandbox.Inbox, func(), error) {
			receivedEmailID = emailID
			return &vaultsandbox.Email{
				Links: []string{"https://example.com"},
			}, nil, func() {}, nil
		}

		cmd := createTestCommand()
		captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Equal(t, "", receivedEmailID)
	})
}

func TestRunURLSpecialCharacters(t *testing.T) {
	t.Run("handles URLs with query parameters", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{
				"https://example.com/verify?token=abc123&user=test@email.com",
			},
		}, nil)

		cmd := createTestCommand()
		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Contains(t, output, "https://example.com/verify?token=abc123&user=test@email.com")
	})

	t.Run("handles URLs with special characters", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: []string{
				"https://example.com/path%20with%20spaces",
				"https://example.com/path?redirect=https://other.com",
			},
		}, nil)

		cmd := createTestCommand()
		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Contains(t, output, "1. https://example.com/path%20with%20spaces")
		assert.Contains(t, output, "2. https://example.com/path?redirect=https://other.com")
	})
}

func TestRunURLManyURLs(t *testing.T) {
	t.Run("handles many URLs", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 0
		links := make([]string, 20)
		for i := 0; i < 20; i++ {
			links[i] = "https://example.com/link" + string(rune('0'+i%10))
		}

		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: links,
		}, nil)

		cmd := createTestCommand()
		output := captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Contains(t, output, "1. ")
		assert.Contains(t, output, "20. ")
	})

	t.Run("opens URL at high index", func(t *testing.T) {
		oldFetcher := getEmailByIDOrLatestFunc
		oldOpenURL := openURLInBrowserFunc
		oldURLOpen := urlOpen
		defer resetURLTestState(oldFetcher, oldOpenURL, oldURLOpen)

		urlOpen = 15
		links := make([]string, 20)
		for i := 0; i < 20; i++ {
			links[i] = "https://example.com/link" + string(rune('a'+i))
		}

		getEmailByIDOrLatestFunc = mockEmailFetcher(&vaultsandbox.Email{
			Links: links,
		}, nil)

		var openedURL string
		openURLInBrowserFunc = func(url string) error {
			openedURL = url
			return nil
		}

		cmd := createTestCommand()
		captureURLStdout(t, func() {
			err := runURL(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Equal(t, "https://example.com/linko", openedURL) // 15th (index 14) = 'a' + 14 = 'o'
	})
}
