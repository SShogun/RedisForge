package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Remove any accidental env vars from outer shell
	os.Unsetenv("SERVER_PORT")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}
}

func TestLoad_CustomPort(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("SERVER_PORT", "99999")
	_, err := Load()
	if err == nil {
		t.Fatal("expected validation error for port 99999")
	}
}
