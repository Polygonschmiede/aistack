package diag

import (
	"regexp"
	"strings"
)

// Redactor handles sensitive data redaction from text
type Redactor struct {
	patterns []redactionPattern
}

type redactionPattern struct {
	regex       *regexp.Regexp
	replacement string
}

// NewRedactor creates a new redactor with common secret patterns
func NewRedactor() *Redactor {
	return &Redactor{
		patterns: []redactionPattern{
			// Environment variables with secrets (must come first)
			{
				regex:       regexp.MustCompile(`(?i)export\s+([A-Z_]*(?:KEY|TOKEN|SECRET|PASSWORD)[A-Z_]*)\s*=\s*["']?([^"'\s]+)["']?`),
				replacement: `export $1=[REDACTED]`,
			},
			// API keys and tokens (capture any preceding character to preserve it)
			{
				regex:       regexp.MustCompile(`(?i)(^|[^A-Z_])(api[_-]?key|token|secret|password)\s*[:=]\s*["']?([^"'\s]+)["']?`),
				replacement: `$1$2: [REDACTED]`,
			},
			// YAML-style secrets
			{
				regex:       regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password):\s*(.+)`),
				replacement: `$1: [REDACTED]`,
			},
			// Bearer tokens
			{
				regex:       regexp.MustCompile(`(?i)Bearer\s+([A-Za-z0-9_\-\.]+)`),
				replacement: `Bearer [REDACTED]`,
			},
			// Basic auth credentials
			{
				regex:       regexp.MustCompile(`(?i)Authorization:\s*Basic\s+([A-Za-z0-9+/=]+)`),
				replacement: `Authorization: Basic [REDACTED]`,
			},
			// Connection strings with passwords
			{
				regex:       regexp.MustCompile(`(?i)(postgres|mysql|mongodb)://([^:]+):([^@]+)@`),
				replacement: `$1://$2:[REDACTED]@`,
			},
		},
	}
}

// Redact applies all redaction patterns to the input text
func (r *Redactor) Redact(input string) string {
	result := input
	for _, pattern := range r.patterns {
		result = pattern.regex.ReplaceAllString(result, pattern.replacement)
	}
	return result
}

// RedactFile reads a file, redacts secrets, and returns the redacted content
func (r *Redactor) RedactFile(content string) string {
	return r.Redact(content)
}

// IsLikelySensitive checks if a line contains potentially sensitive data
func IsLikelySensitive(line string) bool {
	lowerLine := strings.ToLower(line)
	sensitiveKeywords := []string{
		"password", "secret", "token", "api_key", "apikey",
		"private_key", "privatekey", "credential", "auth",
	}

	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lowerLine, keyword) {
			return true
		}
	}
	return false
}
