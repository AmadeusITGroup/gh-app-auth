package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

// sensitiveKeys defines keys that should be redacted in logs
var sensitiveKeys = map[string]bool{
	"token":       true,
	"password":    true,
	"secret":      true,
	"key":         true,
	"private_key": true,
	"pat":         true,
	"credential":  true,
}

// sanitizeValueForLogging checks if a key is sensitive and redacts the value
func sanitizeValueForLogging(key string, value interface{}) interface{} {
	if sensitiveKeys[strings.ToLower(key)] {
		if str, ok := value.(string); ok {
			return RedactToken(str)
		}
		return "<redacted>"
	}
	return value
}

// RedactToken creates a safe redacted representation of a token for logging.
// This is NOT for cryptographic purposes - it's only for log identification.
// It shows length and a prefix to help with debugging without exposing the full token.
func RedactToken(token string) string {
	if token == "" {
		return "empty"
	}

	// Show token length and first 4 chars (if long enough) for identification
	// This is safe for debugging and doesn't expose the full secret
	if len(token) <= 8 {
		return fmt.Sprintf("<redacted:%d>", len(token))
	}
	return fmt.Sprintf("%s...<%d chars>", token[:4], len(token))
}

// HashToken creates a safe hash of a token for logging.
// Deprecated: Use RedactToken instead. This function is kept for backward compatibility.
// Note: This uses SHA256 for fingerprinting only, not for password storage.
func HashToken(token string) string {
	if token == "" {
		return "empty"
	}

	// Use a simple non-cryptographic fingerprint for logging identification
	// We just need a consistent identifier, not cryptographic security
	return RedactToken(token)
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
