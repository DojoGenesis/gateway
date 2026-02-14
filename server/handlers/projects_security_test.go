package handlers

import (
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal name",
			input:    "MyProject",
			expected: "MyProject",
		},
		{
			name:     "Name with spaces",
			input:    "My Project Name",
			expected: "My_Project_Name",
		},
		{
			name:     "Name with special characters",
			input:    "Project@2024!",
			expected: "Project_2024_",
		},
		{
			name:     "Name with path separators",
			input:    "../../../etc/passwd",
			expected: "_________etc_passwd",
		},
		{
			name:     "Name with quotes",
			input:    `Project"With'Quotes`,
			expected: "Project_With_Quotes",
		},
		{
			name:     "Name with Unicode",
			input:    "Project名前",
			expected: "Project名前",
		},
		{
			name:     "Name with hyphens and underscores",
			input:    "My-Cool_Project",
			expected: "My-Cool_Project",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only special characters",
			input:    "!@#$%^&*()",
			expected: "__________",
		},
		{
			name:     "Newlines and tabs",
			input:    "Project\nWith\tWhitespace",
			expected: "Project_With_Whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilenameSecurity(t *testing.T) {
	dangerousInputs := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"file\x00name.txt",
		"<script>alert('xss')</script>",
		"file; rm -rf /",
		"$(whoami)",
	}

	for _, input := range dangerousInputs {
		result := sanitizeFilename(input)

		if result == input {
			t.Errorf("sanitizeFilename(%q) returned unchanged, should have been sanitized", input)
		}

		if containsDangerousCharacters(result) {
			t.Errorf("sanitizeFilename(%q) = %q still contains dangerous characters", input, result)
		}
	}
}

func containsDangerousCharacters(s string) bool {
	dangerous := []string{"..", "/", "\\", "\x00", "<", ">", "|", ";", "$", "(", ")"}
	for _, d := range dangerous {
		if len(d) > 0 && len(s) > 0 {
			for i := 0; i <= len(s)-len(d); i++ {
				if s[i:i+len(d)] == d {
					return true
				}
			}
		}
	}
	return false
}
