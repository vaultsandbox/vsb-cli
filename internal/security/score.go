package security

import (
	"strings"

	vaultsandbox "github.com/vaultsandbox/client-go"
)

// CalculateScore computes a security score (0-100) for an email based on
// authentication results. Base score of 50 assumes E2E encryption.
func CalculateScore(email *vaultsandbox.Email) int {
	score := 50 // Base score for E2E encryption

	if email.AuthResults == nil {
		return score
	}

	auth := email.AuthResults

	if auth.SPF != nil && strings.EqualFold(auth.SPF.Status, "pass") {
		score += 15
	}

	if len(auth.DKIM) > 0 && strings.EqualFold(auth.DKIM[0].Status, "pass") {
		score += 20
	}

	if auth.DMARC != nil && strings.EqualFold(auth.DMARC.Status, "pass") {
		score += 10
	}

	if auth.ReverseDNS != nil && strings.EqualFold(auth.ReverseDNS.Status(), "pass") {
		score += 5
	}

	return score
}
