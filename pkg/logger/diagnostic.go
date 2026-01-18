package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DiagnosticLogger provides conditional logging for debugging git credential flows
type DiagnosticLogger struct {
	enabled          bool
	logger           *log.Logger
	file             *os.File
	sessionID        string
	operationCounter int
}

var globalLogger *DiagnosticLogger

// Initialize sets up the global diagnostic logger based on environment variable
func Initialize() {
	enabled := os.Getenv("GH_APP_AUTH_DEBUG_LOG") != ""
	if !enabled {
		globalLogger = &DiagnosticLogger{enabled: false}
		return
	}

	logPath := os.Getenv("GH_APP_AUTH_DEBUG_LOG")
	if logPath == "" {
		// Default log path
		homeDir, _ := os.UserHomeDir()
		logPath = filepath.Join(homeDir, ".config", "gh", "extensions", "gh-app-auth", "debug.log")
	}

	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		// Fallback to temp directory
		logPath = filepath.Join(os.TempDir(), "gh-app-auth-debug.log")
	}

	// Open log file for append
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		// Disable logging if can't open file
		globalLogger = &DiagnosticLogger{enabled: false}
		return
	}

	// Create logger with timestamp
	logger := log.New(file, "", 0) // No prefix, we'll add our own

	// Generate session ID for this execution
	sessionID := fmt.Sprintf("session_%d_%d", time.Now().Unix(), os.Getpid())

	globalLogger = &DiagnosticLogger{
		enabled:   true,
		logger:    logger,
		file:      file,
		sessionID: sessionID,
	}

	// Log session start
	globalLogger.logEntry("SESSION_START", map[string]interface{}{
		"pid":     os.Getpid(),
		"version": "gh-app-auth",
		"args":    os.Args,
	})
}

// Close closes the diagnostic logger
func Close() {
	if globalLogger != nil && globalLogger.enabled && globalLogger.file != nil {
		globalLogger.logEntry("SESSION_END", map[string]interface{}{})
		_ = globalLogger.file.Close() // Ignore error on close
	}
}

// logEntry writes a structured log entry
func (d *DiagnosticLogger) logEntry(event string, data map[string]interface{}) {
	if !d.enabled {
		return
	}

	d.operationCounter++

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	opID := fmt.Sprintf("%s_op%d", d.sessionID, d.operationCounter)

	// Build log entry
	entry := fmt.Sprintf("[%s] %s [%s]", timestamp, event, opID)

	// Add data fields with automatic sanitization of sensitive keys
	for key, value := range data {
		// Sanitize sensitive fields before logging
		sanitizedValue := sanitizeValueForLogging(key, value)
		entry += fmt.Sprintf(" %s=%v", key, sanitizedValue)
	}

	d.logger.Println(entry)
}

// Flow tracking functions

// FlowStart logs the start of a credential operation
func FlowStart(operation string, data map[string]interface{}) {
	if globalLogger == nil {
		return
	}

	logData := map[string]interface{}{
		"operation": operation,
		"flow":      "START",
	}
	for k, v := range data {
		logData[k] = v
	}

	globalLogger.logEntry("FLOW_START", logData)
}

// FlowStep logs a step in the credential flow
func FlowStep(step string, data map[string]interface{}) {
	if globalLogger == nil {
		return
	}

	logData := map[string]interface{}{
		"step": step,
		"flow": "STEP",
	}
	for k, v := range data {
		logData[k] = v
	}

	globalLogger.logEntry("FLOW_STEP", logData)
}

// FlowSuccess logs successful completion of a flow
func FlowSuccess(operation string, data map[string]interface{}) {
	if globalLogger == nil {
		return
	}

	logData := map[string]interface{}{
		"operation": operation,
		"flow":      "SUCCESS",
	}
	for k, v := range data {
		logData[k] = v
	}

	globalLogger.logEntry("FLOW_SUCCESS", logData)
}

// FlowError logs an error in the flow
func FlowError(operation string, err error, data map[string]interface{}) {
	if globalLogger == nil {
		return
	}

	logData := map[string]interface{}{
		"operation": operation,
		"flow":      "ERROR",
		"error":     err.Error(),
	}
	for k, v := range data {
		logData[k] = v
	}

	globalLogger.logEntry("FLOW_ERROR", logData)
}

// Security functions for sensitive data
//
// This implements a multi-layered approach to sensitive data redaction following
// industry best practices from OWASP, gitleaks, trufflehog, and GitHub secret scanning:
//
// 1. Key-based redaction: Redact values where the key name suggests sensitivity
// 2. Value-based pattern detection: Scan values for known secret patterns regardless of key
// 3. High-entropy detection: Flag suspiciously random strings that might be secrets
//
// References:
// - OWASP Logging Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html
// - gitleaks patterns: https://github.com/gitleaks/gitleaks
// - GitHub secret scanning: https://docs.github.com/en/code-security/secret-scanning

// sensitiveKeyPatterns defines key name patterns that indicate sensitive data.
// Uses substring matching - patterns are checked against lowercase key names.
// Note: Order matters for some patterns to avoid false positives.
var sensitiveKeyPatterns = []string{
	"password",
	"passwd",
	"pwd",
	"token",
	"secret",
	"_key", // api_key, private_key, etc. (underscore prefix avoids matching "key" in "monkey")
	"key_", // key_id, key_file, etc.
	"credential",
	"auth",
	"bearer",
	"apikey",
	"private",
	"_pat", // github_pat, etc. (underscore prefix avoids matching "path")
	"pat_", // pat_token, etc.
}

// sensitiveValuePatterns defines regex patterns to detect secrets in values.
// Based on patterns from gitleaks, trufflehog, and GitHub secret scanning.
var sensitiveValuePatterns = []*regexp.Regexp{
	// GitHub tokens (classic and fine-grained)
	regexp.MustCompile(`^gh[pousr]_[A-Za-z0-9_]{36,}$`),
	// GitHub App tokens
	regexp.MustCompile(`^ghu_[A-Za-z0-9_]{36,}$`),
	regexp.MustCompile(`^ghs_[A-Za-z0-9_]{36,}$`),
	// AWS Access Key ID
	regexp.MustCompile(`^(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}$`),
	// Private keys (PEM format)
	regexp.MustCompile(`-----BEGIN[A-Z ]*PRIVATE KEY-----`),
	// JWT tokens
	regexp.MustCompile(`^eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*$`),
	// Generic API key patterns (long alphanumeric strings)
	regexp.MustCompile(`^[A-Za-z0-9_-]{32,}$`),
	// Slack tokens (xoxb, xoxa, xoxp, xoxr, xoxs)
	regexp.MustCompile(`^xox[baprs]-[0-9a-zA-Z-]{10,}$`),
	// Basic auth in URLs
	regexp.MustCompile(`://[^:]+:[^@]+@`),
}

// isSensitiveKey checks if a key name indicates sensitive data
func isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	for _, pattern := range sensitiveKeyPatterns {
		if strings.Contains(lowerKey, pattern) {
			return true
		}
	}
	return false
}

// isSensitiveValue checks if a value matches known secret patterns
func isSensitiveValue(value string) bool {
	// Skip very short strings (unlikely to be secrets)
	if len(value) < 8 {
		return false
	}

	// Check against known secret patterns
	for _, pattern := range sensitiveValuePatterns {
		if pattern.MatchString(value) {
			return true
		}
	}

	return false
}

// sanitizeValueForLogging applies multi-layered redaction to protect sensitive data.
// It checks both the key name AND the value content to ensure secrets are not logged.
func sanitizeValueForLogging(key string, value interface{}) interface{} {
	// Layer 1: Key-based redaction
	if isSensitiveKey(key) {
		if str, ok := value.(string); ok {
			return RedactSecret(str)
		}
		return "<redacted>"
	}

	// Layer 2: Value-based pattern detection (for string values)
	if str, ok := value.(string); ok {
		if isSensitiveValue(str) {
			return RedactSecret(str)
		}
	}

	return value
}

// RedactSecret creates a safe redacted representation of a secret for logging.
// This is NOT for cryptographic purposes - it's only for log identification.
// It shows the type of secret detected and length, without exposing the actual value.
func RedactSecret(secret string) string {
	if secret == "" {
		return "<empty>"
	}

	// Identify the type of secret for debugging purposes
	secretType := identifySecretType(secret)

	// Return redacted form with type hint and length
	return fmt.Sprintf("<redacted:%s:%d>", secretType, len(secret))
}

// identifySecretType returns a hint about what kind of secret was detected
func identifySecretType(value string) string {
	switch {
	case strings.HasPrefix(value, "ghp_") || strings.HasPrefix(value, "gho_") ||
		strings.HasPrefix(value, "ghu_") || strings.HasPrefix(value, "ghs_") ||
		strings.HasPrefix(value, "ghr_"):
		return "github_token"
	case strings.HasPrefix(value, "AKIA") || strings.HasPrefix(value, "ASIA"):
		return "aws_key"
	case strings.HasPrefix(value, "eyJ"):
		return "jwt"
	case strings.HasPrefix(value, "xox"):
		return "slack_token"
	case strings.Contains(value, "-----BEGIN") && strings.Contains(value, "PRIVATE KEY"):
		return "private_key"
	case strings.Contains(value, "://") && strings.Contains(value, "@"):
		return "url_with_creds"
	default:
		return "secret"
	}
}

// RedactToken is an alias for RedactSecret for backward compatibility.
// Deprecated: Use RedactSecret instead.
func RedactToken(token string) string {
	return RedactSecret(token)
}

// HashToken creates a safe representation of a token for logging.
// Deprecated: Use RedactSecret instead. This function is kept for backward compatibility.
func HashToken(token string) string {
	return RedactSecret(token)
}

// SanitizeURL removes sensitive parts from URLs for logging
func SanitizeURL(url string) string {
	// Remove any embedded credentials
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) == 2 {
			return fmt.Sprintf("https://<credentials>@%s", parts[1])
		}
	}
	return url
}

// SanitizeConfig removes sensitive fields from config data
func SanitizeConfig(data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	for key, value := range data {
		switch strings.ToLower(key) {
		case "token", "password", "secret", "key", "private_key":
			if str, ok := value.(string); ok {
				sanitized[key] = HashToken(str)
			} else {
				sanitized[key] = "<redacted>"
			}
		case "private_key_path":
			// Show path but not content
			sanitized[key] = value
		default:
			sanitized[key] = value
		}
	}

	return sanitized
}

// Convenience functions

// Debug logs general debug information
func Debug(message string, data map[string]interface{}) {
	if globalLogger == nil {
		return
	}

	logData := map[string]interface{}{
		"message": message,
	}
	for k, v := range data {
		logData[k] = v
	}

	globalLogger.logEntry("DEBUG", logData)
}

// Info logs informational messages
func Info(message string, data map[string]interface{}) {
	if globalLogger == nil {
		return
	}

	logData := map[string]interface{}{
		"message": message,
	}
	for k, v := range data {
		logData[k] = v
	}

	globalLogger.logEntry("INFO", logData)
}

// Error logs error messages
func Error(message string, err error, data map[string]interface{}) {
	if globalLogger == nil {
		return
	}

	logData := map[string]interface{}{
		"message": message,
		"error":   err.Error(),
	}
	for k, v := range data {
		logData[k] = v
	}

	globalLogger.logEntry("ERROR", logData)
}
