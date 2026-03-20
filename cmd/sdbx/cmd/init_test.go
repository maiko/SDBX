package cmd

import (
	"errors"
	"testing"
)

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "Hello"},
		{"Hello", "Hello"},
		{"", ""},
		{"a", "A"},
		{"UPPER", "UPPER"},
		{"media", "Media"},
		{"downloads", "Downloads"},
	}

	for _, tt := range tests {
		result := capitalizeFirst(tt.input)
		if result != tt.expected {
			t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestValidateAdminPassword(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"", true},
		{"short", true},
		{"seven7!", true},
		{"eightchr", false},
		{"longenoughpassword", false},
		{"12345678", false},
	}

	for _, tt := range tests {
		err := validateAdminPassword(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateAdminPassword(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

func TestErrStartOver(t *testing.T) {
	// Verify errStartOver is a distinct sentinel error
	if errStartOver == nil {
		t.Fatal("errStartOver should not be nil")
	}
	if !errors.Is(errStartOver, errStartOver) {
		t.Error("errStartOver should match itself with errors.Is")
	}
}
