package auth

import (
	"net/http"
	"strings"
	"testing"
)

func TestFormatAPIStatusError(t *testing.T) {
	// The exact body GitHub returns when the local clock runs ahead of theirs.
	clockSkewBody := `{"message":"'Expiration time' claim ('exp') is too far in the future",` +
		`"documentation_url":"https://docs.github.com/rest","status":"401"}`

	tests := []struct {
		name     string
		status   int
		body     string
		wantHint bool
	}{
		{"exp too far in the future", http.StatusUnauthorized, clockSkewBody, true},
		{
			"iat in the future",
			http.StatusUnauthorized,
			`{"message":"'Issued at' claim ('iat') is in the future","status":"401"}`,
			true,
		},
		{
			"unrelated 401",
			http.StatusUnauthorized,
			`{"message":"A JSON web token could not be decoded","status":"401"}`,
			false,
		},
		{"not found", http.StatusNotFound, `{"message":"Not Found","status":"404"}`, false},
		{"forbidden", http.StatusForbidden, `{"message":"Resource not accessible","status":"403"}`, false},
		{"empty body", http.StatusUnauthorized, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FormatAPIStatusError(tt.status, []byte(tt.body))
			if err == nil {
				t.Fatal("FormatAPIStatusError() returned nil, want error")
			}

			got := err.Error()

			// The status code and the raw API body must always survive.
			if !strings.Contains(got, tt.body) {
				t.Errorf("error %q should contain the API body %q", got, tt.body)
			}
			if !strings.Contains(got, http.StatusText(tt.status)) && !strings.Contains(got, "status") {
				t.Errorf("error %q should mention the status", got)
			}

			hasHint := strings.Contains(got, "synchronize the system clock")
			if hasHint != tt.wantHint {
				t.Errorf("clock skew hint present = %v, want %v (error: %q)", hasHint, tt.wantHint, got)
			}
		})
	}
}

func TestIsClockSkewMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{"exp claim", "'Expiration time' claim ('exp') is too far in the future", true},
		{"iat claim", "'Issued at' claim ('iat') is in the future", true},
		{"case insensitive", "'EXPIRATION TIME' CLAIM ('EXP') IS TOO FAR IN THE FUTURE", true},
		{"bad credentials", "Bad credentials", false},
		{"empty", "", false},
		{"expired token", "token expired", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClockSkewMessage(tt.message); got != tt.want {
				t.Errorf("isClockSkewMessage(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}
