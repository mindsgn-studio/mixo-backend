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

func TestConfigAuthDefaults(t *testing.T) {
	unsetEnv(t, "ADMIN_USERNAME", "ADMIN_PASSWORD")
	cfg := &Config{
		AdminUsername: "admin",
		AdminPassword: "",
	}
	if cfg.AdminUsername != "admin" {
		t.Errorf("Expected default admin username 'admin', got %q", cfg.AdminUsername)
	}
	if cfg.AdminPassword != "" {
		t.Errorf("Expected empty admin password by default, got %q", cfg.AdminPassword)
	}
}

func TestLoadAdminAuthFromEnv(t *testing.T) {
	unsetEnv(t, "ADMIN_USERNAME", "ADMIN_PASSWORD")

	if err := os.Setenv("ADMIN_USERNAME", "radio"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("ADMIN_PASSWORD", "hunter2"); err != nil {
		t.Fatal(err)
	}
	defer unsetEnv(t, "ADMIN_USERNAME", "ADMIN_PASSWORD")

	cfg := &Config{
		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
	}
	if cfg.AdminUsername != "radio" {
		t.Errorf("Expected admin username 'radio', got %q", cfg.AdminUsername)
	}
	if cfg.AdminPassword != "hunter2" {
		t.Errorf("Expected admin password 'hunter2', got %q", cfg.AdminPassword)
	}
}

func TestHLSDefaults(t *testing.T) {
	cfg := &Config{
		HLSDir:          "/var/www/html/hls",
		HLSStreamID:     "radio",
		HLSSegmentTime:  2,
		HLSPlaylistSize: 6,
	}
	if cfg.HLSDir != "/var/www/html/hls" {
		t.Errorf("Expected default HLS_DIR /var/www/html/hls, got %q", cfg.HLSDir)
	}
	if cfg.HLSStreamID != "radio" {
		t.Errorf("Expected default HLS_STREAM_ID 'radio', got %q", cfg.HLSStreamID)
	}
	if cfg.HLSSegmentTime != 2 || cfg.HLSPlaylistSize != 6 {
		t.Errorf("unexpected hls tuning: segment=%d playlist=%d", cfg.HLSSegmentTime, cfg.HLSPlaylistSize)
	}
}

func TestLoadHLSDirFromEnv(t *testing.T) {
	unsetEnv(t, "HLS_DIR")
	if err := os.Setenv("HLS_DIR", "/srv/hls"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Unsetenv("HLS_DIR") }()

	if got := getEnv("HLS_DIR", "/var/www/html/hls"); got != "/srv/hls" {
		t.Errorf("Expected HLS_DIR /srv/hls, got %q", got)
	}
}

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("Failed to unset %s: %v", k, err)
		}
	}
}
