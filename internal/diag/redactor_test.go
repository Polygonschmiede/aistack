package diag

import (
	"strings"
	"testing"
)

func TestRedactor_Redact(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "API key in config",
			input:    "api_key: sk-1234567890abcdef",
			expected: "api_key: [REDACTED]",
		},
		{
			name:     "Token with quotes",
			input:    `token = "ghp_abc123xyz"`,
			expected: `token: [REDACTED]`,
		},
		{
			name:     "Password in YAML",
			input:    "password: super_secret_123",
			expected: "password: [REDACTED]",
		},
		{
			name:     "Environment variable",
			input:    "export OPENAI_API_KEY=sk-proj-xyz123",
			expected: "export OPENAI_API_KEY=[REDACTED]",
		},
		{
			name:     "Bearer token",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Authorization: Bearer [REDACTED]",
		},
		{
			name:     "Basic auth",
			input:    "Authorization: Basic dXNlcjpwYXNzd29yZA==",
			expected: "Authorization: Basic [REDACTED]",
		},
		{
			name:     "Database connection string",
			input:    "postgres://user:password123@localhost:5432/db",
			expected: "postgres://user:[REDACTED]@localhost:5432/db",
		},
		{
			name:     "Non-sensitive data",
			input:    "log_level: debug\nport: 8080",
			expected: "log_level: debug\nport: 8080",
		},
		{
			name:     "Multiple secrets",
			input:    "api_key: sk-123\ntoken: ghp-456\npassword: secret",
			expected: "api_key: [REDACTED]\ntoken: [REDACTED]\npassword: [REDACTED]",
		},
	}

	redactor := NewRedactor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.Redact(tt.input)
			if result != tt.expected {
				t.Errorf("Redact() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRedactor_RedactFile(t *testing.T) {
	redactor := NewRedactor()

	content := `# Config file
log_level: info
api_key: sk-1234567890
database_url: postgres://user:pass123@db:5432/mydb
timeout: 30
`

	redacted := redactor.RedactFile(content)

	// Should not contain secrets
	if strings.Contains(redacted, "sk-1234567890") {
		t.Error("API key was not redacted")
	}
	if strings.Contains(redacted, "pass123") {
		t.Error("Database password was not redacted")
	}

	// Should contain non-sensitive data
	if !strings.Contains(redacted, "log_level: info") {
		t.Error("Non-sensitive config was modified")
	}
	if !strings.Contains(redacted, "timeout: 30") {
		t.Error("Non-sensitive config was modified")
	}

	// Should contain redaction markers
	if !strings.Contains(redacted, "[REDACTED]") {
		t.Error("Redaction markers not present")
	}
}

func TestIsLikelySensitive(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "API key field",
			line:     "api_key: sk-123",
			expected: true,
		},
		{
			name:     "Password field",
			line:     "password: secret",
			expected: true,
		},
		{
			name:     "Token field",
			line:     "token = abc123",
			expected: true,
		},
		{
			name:     "Private key",
			line:     "private_key: -----BEGIN",
			expected: true,
		},
		{
			name:     "Credentials",
			line:     "credentials: {user: admin}",
			expected: true,
		},
		{
			name:     "Auth header",
			line:     "Authorization: Bearer xyz",
			expected: true,
		},
		{
			name:     "Normal config",
			line:     "log_level: debug",
			expected: false,
		},
		{
			name:     "Port number",
			line:     "port: 8080",
			expected: false,
		},
		{
			name:     "Empty line",
			line:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLikelySensitive(tt.line)
			if result != tt.expected {
				t.Errorf("IsLikelySensitive() = %v, want %v", result, tt.expected)
			}
		})
	}
}
