package config

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	// Test with existing env var
	if err := os.Setenv("TEST_VAR", "test_value"); err != nil {
		t.Fatalf("Failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_VAR"); err != nil {
			t.Fatalf("Failed to unset env: %v", err)
		}
	}()

	result := getEnv("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}

	// Test with non-existent env var
	result = getEnv("NON_EXISTENT_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	// Test with valid int env var
	if err := os.Setenv("TEST_INT", "42"); err != nil {
		t.Fatalf("Failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_INT"); err != nil {
			t.Fatalf("Failed to unset env: %v", err)
		}
	}()

	result := getEnvAsInt("TEST_INT", "10")
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	// Test with invalid int env var (should use default)
	if err := os.Setenv("TEST_INVALID", "not_a_number"); err != nil {
		t.Fatalf("Failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_INVALID"); err != nil {
			t.Fatalf("Failed to unset env: %v", err)
		}
	}()

	result = getEnvAsInt("TEST_INVALID", "10")
	if result != 10 {
		t.Errorf("Expected 10 (default), got %d", result)
	}

	// Test with non-existent env var
	result = getEnvAsInt("NON_EXISTENT_INT", "10")
	if result != 10 {
		t.Errorf("Expected 10 (default), got %d", result)
	}
}
