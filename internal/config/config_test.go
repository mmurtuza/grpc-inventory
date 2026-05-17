package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()

	cfg := Load()

	if cfg.RestAPIPort != "8080" {
		t.Errorf("expected 8080, got %q", cfg.RestAPIPort)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("expected text, got %q", cfg.LogFormat)
	}
}

func TestLoad_EnvVars(t *testing.T) {
	os.Clearenv()
	os.Setenv("REST_API_PORT", "9090")
	os.Setenv("LOG_FORMAT", "json")

	cfg := Load()

	if cfg.RestAPIPort != "9090" {
		t.Errorf("expected 9090, got %q", cfg.RestAPIPort)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("expected json, got %q", cfg.LogFormat)
	}
}
