package sanitizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"short", "***"},
		{"medium12", "med***12"},
		{"verylongapikey123456", "ver***456"},
	}

	for _, tt := range tests {
		result := APIKey(tt.input)
		assert.Contains(t, result, "***")
		assert.NotEqual(t, tt.input, result)
	}
}
