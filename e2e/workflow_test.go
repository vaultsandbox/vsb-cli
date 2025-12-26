//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Cross-Command Workflow Tests
// ============================================================================
// These tests verify realistic user scenarios that span multiple commands,
// ensuring commands work together correctly in typical usage patterns.

// TestCIWorkflow simulates a typical CI/CD testing workflow:
// 1. Create inbox for test run
// 2. Trigger action that sends email (simulated via SMTP)
// 3. Wait for email with specific subject
// 4. Extract verification link
// 5. Verify link format
// 6. Cleanup
func TestCIWorkflow(t *testing.T) {
	skipIfNoSMTP(t)
	configDir := t.TempDir()

	// Step 1: Create inbox for this test run
	stdout, stderr, code := runVSBWithConfig(t, configDir, "inbox", "create", "--ttl", "1h", "--output", "json")
	require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	inboxEmail := createResult.Email
	t.Logf("Created CI test inbox: %s", inboxEmail)

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inboxEmail)
	})

	// Step 2: Simulate application sending verification email
	timestamp := time.Now().Format("150405.000")
	verifyToken := "ci-test-token-" + timestamp
	subject := "Welcome to CI Test " + timestamp
	htmlBody := `<!DOCTYPE html>
<html>
<body>
<h1>Welcome!</h1>
<p>Click to verify: <a href="https://app.example.com/verify?token=` + verifyToken + `">Verify Account</a></p>
<p>Or copy this link: https://app.example.com/verify?token=` + verifyToken + `</p>
</body>
</html>`
	textBody := "Welcome! Verify at: https://app.example.com/verify?token=" + verifyToken

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)
		<-sendTestHTMLEmailAsync(inboxEmail, subject, textBody, htmlBody)
	}()

	// Step 3: Wait for email with specific subject pattern
	stdout, stderr, code = runVSBWithConfig(t, configDir, "wait",
		"--subject-regex", "Welcome.*CI Test",
		"--timeout", "30s",
		"--output", "json")
	require.Equal(t, 0, code, "wait failed: stdout=%s, stderr=%s", stdout, stderr)

	var waitResult struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &waitResult))
	assert.Contains(t, waitResult.Subject, "CI Test")

	// Step 4: Extract verification link
	stdout, stderr, code = runVSBWithConfig(t, configDir, "email", "url", waitResult.ID, "--output", "json")
	require.Equal(t, 0, code, "url extraction failed: stdout=%s, stderr=%s", stdout, stderr)

	var links []string
	require.NoError(t, json.Unmarshal([]byte(stdout), &links))

	// Step 5: Verify link format
	var verifyLink string
	for _, link := range links {
		if strings.Contains(link, "verify?token=") {
			verifyLink = link
			break
		}
	}
	require.NotEmpty(t, verifyLink, "verification link not found in email")
	assert.Contains(t, verifyLink, verifyToken)

	t.Logf("CI workflow complete: extracted verify link %s", verifyLink)
	wg.Wait()
}

// TestMultiInboxWorkflow tests managing multiple inboxes simultaneously:
// 1. Create multiple inboxes for different services
// 2. Send emails to each
// 3. Switch between inboxes and verify correct emails
// 4. Export all inboxes
// 5. Cleanup
func TestMultiInboxWorkflow(t *testing.T) {
	skipIfNoSMTP(t)
	configDir := t.TempDir()

	// Step 1: Create inboxes for different "services"
	services := []string{"auth", "billing", "support"}
	inboxes := make(map[string]string) // service -> email

	for _, service := range services {
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		inboxes[service] = result.Email
		t.Logf("Created %s inbox: %s", service, result.Email)
	}

	t.Cleanup(func() {
		for _, email := range inboxes {
			runVSBWithConfig(t, configDir, "inbox", "delete", email)
		}
	})

	// Step 2: Send unique emails to each inbox
	for service, email := range inboxes {
		subject := "Message from " + service + " service"
		body := "This is a test email for " + service
		sendTestEmail(t, email, subject, body)
	}
	time.Sleep(3 * time.Second)

	// Step 3: Switch between inboxes and verify correct emails
	for service, email := range inboxes {
		// Switch to this inbox
		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "use", email)
		require.Equal(t, 0, code, "use failed for %s: stderr=%s", service, stderr)

		// Verify active inbox
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "info", "--output", "json")
		require.Equal(t, 0, code)

		var info struct {
			Email    string `json:"email"`
			IsActive bool   `json:"isActive"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &info))
		assert.Equal(t, email, info.Email)
		assert.True(t, info.IsActive)

		// List emails and verify correct service email is present
		stdout, _, code = runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
		require.Equal(t, 0, code)

		var emails []struct {
			Subject string `json:"subject"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &emails))

		found := false
		expectedSubject := "Message from " + service + " service"
		for _, e := range emails {
			if e.Subject == expectedSubject {
				found = true
				break
			}
		}
		assert.True(t, found, "expected email for %s service not found", service)
	}

	// Step 4: Export all inboxes
	exportDir := t.TempDir()
	for service, email := range inboxes {
		exportPath := filepath.Join(exportDir, service+"-inbox.json")
		_, stderr, code := runVSBWithConfig(t, configDir, "export", email, "--out", exportPath)
		require.Equal(t, 0, code, "export failed for %s: stderr=%s", service, stderr)

		// Verify export file exists
		_, err := os.Stat(exportPath)
		require.NoError(t, err, "export file for %s should exist", service)
	}

	t.Log("Multi-inbox workflow complete")
}

// TestBackupRestoreWorkflow tests a complete backup and restore scenario:
// 1. Create inbox and populate with emails
// 2. Export to backup file
// 3. Simulate "disaster" by deleting local config
// 4. Restore from backup
// 5. Verify all emails are accessible
func TestBackupRestoreWorkflow(t *testing.T) {
	skipIfNoSMTP(t)
	configDir := t.TempDir()

	// Step 1: Create inbox
	stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	inboxEmail := createResult.Email

	// Send multiple emails with distinct content
	testEmails := []struct {
		subject string
		body    string
	}{
		{"Backup Test 1", "First important email"},
		{"Backup Test 2", "Second important email"},
		{"Backup Test 3", "Third important email"},
	}

	for _, e := range testEmails {
		sendTestEmail(t, inboxEmail, e.subject, e.body)
	}
	time.Sleep(3 * time.Second)

	// Verify all emails received
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
	require.Equal(t, 0, code)

	var emailsBefore []struct {
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &emailsBefore))
	require.GreaterOrEqual(t, len(emailsBefore), 3)

	// Step 2: Export to backup
	backupPath := filepath.Join(t.TempDir(), "backup.json")
	_, stderr, code := runVSBWithConfig(t, configDir, "export", "--out", backupPath)
	require.Equal(t, 0, code, "export failed: stderr=%s", stderr)

	// Step 3: Simulate "disaster" - delete local config
	_, _, _ = runVSBWithConfig(t, configDir, "inbox", "delete", "--local", inboxEmail)

	// Verify inbox is gone locally
	stdout, _, code = runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
	require.Equal(t, 0, code)

	var listAfterDelete []struct {
		Email string `json:"email"`
	}
	json.Unmarshal([]byte(stdout), &listAfterDelete)

	found := false
	for _, inbox := range listAfterDelete {
		if inbox.Email == inboxEmail {
			found = true
			break
		}
	}
	assert.False(t, found, "inbox should be deleted locally")

	// Step 4: Restore from backup (using new config dir to simulate new machine)
	restoreConfigDir := t.TempDir()
	_, stderr, code = runVSBWithConfig(t, restoreConfigDir, "import", backupPath)
	require.Equal(t, 0, code, "import failed: stderr=%s", stderr)

	// Step 5: Verify inbox restored
	stdout, _, code = runVSBWithConfig(t, restoreConfigDir, "inbox", "info", "--output", "json")
	require.Equal(t, 0, code)

	var inboxInfo struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &inboxInfo))
	assert.Equal(t, inboxEmail, inboxInfo.Email)

	// Verify all emails still accessible
	stdout, _, code = runVSBWithConfig(t, restoreConfigDir, "email", "list", "--output", "json")
	require.Equal(t, 0, code)

	var emailsAfter []struct {
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &emailsAfter))

	for _, expected := range testEmails {
		found := false
		for _, e := range emailsAfter {
			if e.Subject == expected.subject {
				found = true
				break
			}
		}
		assert.True(t, found, "email with subject %q should be accessible after restore", expected.subject)
	}

	// Cleanup
	t.Cleanup(func() {
		runVSBWithConfig(t, restoreConfigDir, "inbox", "delete", inboxEmail)
	})

	t.Log("Backup/restore workflow complete")
}

// TestPasswordResetWorkflow simulates a password reset flow:
// 1. Create inbox for user
// 2. Trigger password reset (simulated email)
// 3. Wait for reset email with regex pattern
// 4. Extract reset link
// 5. Verify link contains expected token format
func TestPasswordResetWorkflow(t *testing.T) {
	skipIfNoSMTP(t)
	configDir := t.TempDir()

	// Step 1: Create inbox
	stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	inboxEmail := createResult.Email

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inboxEmail)
	})

	// Step 2: Simulate password reset email
	resetToken := "reset-" + time.Now().Format("20060102150405")
	htmlBody := `<!DOCTYPE html>
<html>
<body>
<h1>Password Reset Request</h1>
<p>Click the link below to reset your password:</p>
<a href="https://auth.example.com/reset?token=` + resetToken + `&expires=3600">Reset Password</a>
<p>This link expires in 1 hour.</p>
<p>If you didn't request this, please ignore this email.</p>
</body>
</html>`
	textBody := "Reset your password: https://auth.example.com/reset?token=" + resetToken + "&expires=3600"

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)
		<-sendTestHTMLEmailAsync(inboxEmail, "Password Reset Request", textBody, htmlBody)
	}()

	// Step 3: Wait for reset email
	stdout, stderr, code := runVSBWithConfig(t, configDir, "wait",
		"--subject-regex", "Password.*Reset",
		"--timeout", "30s",
		"--output", "json")
	require.Equal(t, 0, code, "wait failed: stdout=%s, stderr=%s", stdout, stderr)

	var waitResult struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &waitResult))

	// Step 4: Extract reset link using --extract-link
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "url", waitResult.ID, "--output", "json")
	require.Equal(t, 0, code)

	var links []string
	require.NoError(t, json.Unmarshal([]byte(stdout), &links))

	// Step 5: Find and verify reset link
	var resetLink string
	for _, link := range links {
		if strings.Contains(link, "reset?token=") {
			resetLink = link
			break
		}
	}
	require.NotEmpty(t, resetLink, "reset link not found")
	assert.Contains(t, resetLink, resetToken)
	assert.Contains(t, resetLink, "expires=")

	t.Logf("Password reset workflow complete: %s", resetLink)
	wg.Wait()
}

// TestOrderConfirmationWorkflow simulates an e-commerce order confirmation flow:
// 1. Create inbox
// 2. "Place order" (send confirmation email)
// 3. Wait for order confirmation
// 4. View full email details
// 5. Audit email for security
// 6. Extract any tracking links
func TestOrderConfirmationWorkflow(t *testing.T) {
	skipIfNoSMTP(t)
	configDir := t.TempDir()

	// Step 1: Create inbox
	stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	inboxEmail := createResult.Email

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inboxEmail)
	})

	// Step 2: Simulate order confirmation
	orderID := "ORD-" + time.Now().Format("20060102-150405")
	htmlBody := `<!DOCTYPE html>
<html>
<body>
<h1>Order Confirmation</h1>
<p>Thank you for your order!</p>
<table>
<tr><td>Order ID:</td><td>` + orderID + `</td></tr>
<tr><td>Total:</td><td>$99.99</td></tr>
<tr><td>Status:</td><td>Processing</td></tr>
</table>
<p><a href="https://shop.example.com/orders/` + orderID + `">View Order</a></p>
<p><a href="https://shipping.example.com/track/` + orderID + `">Track Shipment</a></p>
</body>
</html>`
	textBody := "Order Confirmation\nOrder ID: " + orderID + "\nTotal: $99.99\nView: https://shop.example.com/orders/" + orderID

	sendTestHTMLEmail(t, inboxEmail, "Order Confirmation - "+orderID, textBody, htmlBody)
	time.Sleep(2 * time.Second)

	// Step 3: Wait for confirmation
	stdout, stderr, code := runVSBWithConfig(t, configDir, "wait",
		"--subject-regex", "Order Confirmation.*"+orderID[:15],
		"--timeout", "30s",
		"--output", "json")
	require.Equal(t, 0, code, "wait failed: stdout=%s, stderr=%s", stdout, stderr)

	var waitResult struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &waitResult))

	// Step 4: View full email
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "view", waitResult.ID, "--output", "json")
	require.Equal(t, 0, code)

	var emailView struct {
		Subject string `json:"subject"`
		HTML    string `json:"html"`
		Text    string `json:"text"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &emailView))
	assert.Contains(t, emailView.Subject, orderID)

	// Step 5: Audit email security
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "audit", waitResult.ID, "--output", "json")
	require.Equal(t, 0, code)

	var audit struct {
		SecurityScore int `json:"securityScore"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &audit))
	assert.GreaterOrEqual(t, audit.SecurityScore, 0)

	// Step 6: Extract links
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "url", waitResult.ID, "--output", "json")
	require.Equal(t, 0, code)

	var links []string
	require.NoError(t, json.Unmarshal([]byte(stdout), &links))

	// Verify order and tracking links found
	var foundOrder, foundTrack bool
	for _, link := range links {
		if strings.Contains(link, "orders/"+orderID) {
			foundOrder = true
		}
		if strings.Contains(link, "track/"+orderID) {
			foundTrack = true
		}
	}
	assert.True(t, foundOrder, "should find order link")
	assert.True(t, foundTrack, "should find tracking link")

	t.Logf("Order confirmation workflow complete for %s", orderID)
}

// TestInboxSharingWorkflow tests sharing an inbox between users/machines:
// 1. Create inbox on "machine A"
// 2. Send some emails
// 3. Export inbox
// 4. Import on "machine B" (different config dir)
// 5. Both can access emails
// 6. Send more emails, verify visible on both
func TestInboxSharingWorkflow(t *testing.T) {
	skipIfNoSMTP(t)

	// Machine A setup
	machineA := t.TempDir()
	stdout, _, code := runVSBWithConfig(t, machineA, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	sharedEmail := createResult.Email

	t.Cleanup(func() {
		runVSBWithConfig(t, machineA, "inbox", "delete", sharedEmail)
	})

	// Send initial emails from "machine A perspective"
	sendTestEmail(t, sharedEmail, "From Machine A", "First email sent")
	time.Sleep(2 * time.Second)

	// Export inbox
	exportPath := filepath.Join(t.TempDir(), "shared-inbox.json")
	_, stderr, code := runVSBWithConfig(t, machineA, "export", "--out", exportPath)
	require.Equal(t, 0, code, "export failed: stderr=%s", stderr)

	// Machine B setup - import the shared inbox
	machineB := t.TempDir()
	_, stderr, code = runVSBWithConfig(t, machineB, "import", exportPath)
	require.Equal(t, 0, code, "import on machine B failed: stderr=%s", stderr)

	// Verify machine B can see the email
	stdout, _, code = runVSBWithConfig(t, machineB, "email", "list", "--output", "json")
	require.Equal(t, 0, code)

	var emailsOnB []struct {
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &emailsOnB))

	foundInitial := false
	for _, e := range emailsOnB {
		if e.Subject == "From Machine A" {
			foundInitial = true
			break
		}
	}
	assert.True(t, foundInitial, "machine B should see email sent before import")

	// Send new email (both machines should see it)
	sendTestEmail(t, sharedEmail, "After Sharing", "This email sent after sharing")
	time.Sleep(2 * time.Second)

	// Verify on machine A
	stdout, _, code = runVSBWithConfig(t, machineA, "email", "list", "--output", "json")
	require.Equal(t, 0, code)

	var emailsOnA []struct {
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &emailsOnA))

	foundNew := false
	for _, e := range emailsOnA {
		if e.Subject == "After Sharing" {
			foundNew = true
			break
		}
	}
	assert.True(t, foundNew, "machine A should see new email")

	// Verify on machine B
	stdout, _, code = runVSBWithConfig(t, machineB, "email", "list", "--output", "json")
	require.Equal(t, 0, code)

	require.NoError(t, json.Unmarshal([]byte(stdout), &emailsOnB))

	foundNew = false
	for _, e := range emailsOnB {
		if e.Subject == "After Sharing" {
			foundNew = true
			break
		}
	}
	assert.True(t, foundNew, "machine B should see new email")

	t.Log("Inbox sharing workflow complete")
}

// TestWaitThenProcessWorkflow tests the wait-then-process pattern:
// 1. Create inbox
// 2. Start wait in background (simulated by goroutine)
// 3. Send email
// 4. Verify wait completes
// 5. Process the returned email ID
func TestWaitThenProcessWorkflow(t *testing.T) {
	skipIfNoSMTP(t)
	configDir := t.TempDir()

	// Create inbox
	stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	inboxEmail := createResult.Email

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inboxEmail)
	})

	timestamp := time.Now().Format("150405.000")
	uniqueSubject := "Process Me " + timestamp

	// Send email after delay
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond)
		<-sendTestEmailAsync(inboxEmail, uniqueSubject, "Content to process")
	}()

	// Wait for the specific email
	stdout, stderr, code := runVSBWithConfig(t, configDir, "wait",
		"--subject", uniqueSubject,
		"--timeout", "30s",
		"--output", "json")
	require.Equal(t, 0, code, "wait failed: stdout=%s, stderr=%s", stdout, stderr)

	var waitResult struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
		Text    string `json:"text"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &waitResult))
	assert.Equal(t, uniqueSubject, waitResult.Subject)

	// Process: View full details
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "view", waitResult.ID, "--output", "json")
	require.Equal(t, 0, code)

	var viewResult struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &viewResult))
	assert.Equal(t, waitResult.ID, viewResult.ID)

	// Process: Audit
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "audit", waitResult.ID, "--output", "json")
	require.Equal(t, 0, code)

	var auditResult struct {
		ID            string `json:"id"`
		SecurityScore int    `json:"securityScore"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &auditResult))
	assert.Equal(t, waitResult.ID, auditResult.ID)

	wg.Wait()
	t.Log("Wait-then-process workflow complete")
}

// TestBulkEmailProcessingWorkflow tests processing multiple emails in batch:
// 1. Create inbox
// 2. Send multiple emails
// 3. List all emails
// 4. Process each email (view, extract URLs)
// 5. Delete processed emails
func TestBulkEmailProcessingWorkflow(t *testing.T) {
	skipIfNoSMTP(t)
	configDir := t.TempDir()

	// Create inbox
	stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	inboxEmail := createResult.Email

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inboxEmail)
	})

	// Send multiple emails
	timestamp := time.Now().Format("150405")
	subjects := []string{
		"Bulk Process 1 - " + timestamp,
		"Bulk Process 2 - " + timestamp,
		"Bulk Process 3 - " + timestamp,
	}

	for i, subject := range subjects {
		body := `Visit our site: https://example.com/page` + string(rune('1'+i))
		sendTestEmail(t, inboxEmail, subject, body)
	}
	time.Sleep(3 * time.Second)

	// List all emails
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
	require.Equal(t, 0, code)

	var emails []struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &emails))

	// Filter to our bulk emails
	var bulkEmails []struct {
		ID      string
		Subject string
	}
	for _, e := range emails {
		if strings.HasPrefix(e.Subject, "Bulk Process") && strings.Contains(e.Subject, timestamp) {
			bulkEmails = append(bulkEmails, struct {
				ID      string
				Subject string
			}{e.ID, e.Subject})
		}
	}
	require.Equal(t, 3, len(bulkEmails), "should find all 3 bulk emails")

	// Process each email
	for _, e := range bulkEmails {
		// View
		stdout, _, code = runVSBWithConfig(t, configDir, "email", "view", e.ID, "--output", "json")
		require.Equal(t, 0, code)

		// Extract URLs
		stdout, _, code = runVSBWithConfig(t, configDir, "email", "url", e.ID, "--output", "json")
		require.Equal(t, 0, code)

		var links []string
		require.NoError(t, json.Unmarshal([]byte(stdout), &links))
		assert.NotEmpty(t, links, "each email should have a URL")
	}

	// Delete all processed emails
	for _, e := range bulkEmails {
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "delete", e.ID)
		require.Equal(t, 0, code, "delete failed for %s: stderr=%s", e.ID, stderr)
	}

	// Verify all deleted
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
	require.Equal(t, 0, code)

	require.NoError(t, json.Unmarshal([]byte(stdout), &emails))

	for _, e := range bulkEmails {
		found := false
		for _, remaining := range emails {
			if remaining.ID == e.ID {
				found = true
				break
			}
		}
		assert.False(t, found, "email %s should be deleted", e.ID)
	}

	t.Log("Bulk email processing workflow complete")
}
