package auth

import (
	"fmt"
	"net/http"
	"strings"
)

// clockSkewHint is appended to GitHub 401 responses that indicate the JWT
// timestamps were rejected. These failures are caused by the local clock
// disagreeing with GitHub's, not by an invalid App configuration, so the raw
// API message on its own tends to send users looking in the wrong place.
const clockSkewHint = "the JWT timestamps were rejected because this machine's clock disagrees with GitHub's; " +
	"synchronize the system clock (e.g. enable NTP on the host or container) and retry"

// FormatAPIStatusError builds an error for a non-success GitHub API response,
// adding actionable context for failure modes whose raw message is misleading.
func FormatAPIStatusError(statusCode int, body []byte) error {
	message := string(body)

	if statusCode == http.StatusUnauthorized && isClockSkewMessage(message) {
		return fmt.Errorf("GitHub API returned status %d: %s (%s)", statusCode, message, clockSkewHint)
	}

	return fmt.Errorf("GitHub API returned status %d: %s", statusCode, message)
}

// isClockSkewMessage reports whether a GitHub error body describes a JWT
// timestamp rejection. GitHub phrases these as complaints about the 'exp' or
// 'iat' claims.
func isClockSkewMessage(message string) bool {
	lower := strings.ToLower(message)

	skewPhrases := []string{
		"'exp') is too far in the future",
		"'iat') is in the future",
		"expiration time' claim ('exp') is too far in the future",
		"issued at' claim ('iat') is in the future",
	}

	for _, phrase := range skewPhrases {
		if strings.Contains(lower, strings.ToLower(phrase)) {
			return true
		}
	}

	return false
}
