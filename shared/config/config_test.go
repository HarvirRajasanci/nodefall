package config

import (
	"errors"
	"os"
	"testing"
)

func TestLoad_ReadsSetValues(t *testing.T) {
	os.Setenv("NODEFALL_JWT_SECRET", "test-secret")
	os.Setenv("NODEFALL_REDIS_URL", "redis://localhost:6379")
	defer os.Unsetenv("NODEFALL_JWT_SECRET")
	defer os.Unsetenv("NODEFALL_REDIS_URL")

	cfg := Load()

	if cfg.JWTSecret != "test-secret" {
		t.Errorf("got JWTSecret %q, want %q", cfg.JWTSecret, "test-secret")
	}
	if cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("got RedisURL %q, want %q", cfg.RedisURL, "redis://localhost:6379")
	}
}

func TestLoad_LeavesUnsetValuesEmpty(t *testing.T) {
	os.Unsetenv("NODEFALL_DB_URL")

	cfg := Load()

	if cfg.DBURL != "" {
		t.Errorf("got DBURL %q, want empty string", cfg.DBURL)
	}
}

func TestRequireJWTSecret_ReturnsSecretWhenSet(t *testing.T) {
	cfg := Config{JWTSecret: "test-secret"}

	secret, err := cfg.RequireJWTSecret()
	if err != nil {
		t.Fatalf("RequireJWTSecret returned error: %v", err)
	}
	if string(secret) != "test-secret" {
		t.Errorf("got %q, want %q", secret, "test-secret")
	}
}

func TestRequireJWTSecret_ErrorsWhenEmpty(t *testing.T) {
	cfg := Config{}

	if _, err := cfg.RequireJWTSecret(); !errors.Is(err, ErrMissingJWTSecret) {
		t.Errorf("got err %v, want ErrMissingJWTSecret", err)
	}
}
