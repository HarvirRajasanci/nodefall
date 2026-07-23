// Package config centralizes loading of environment-configured settings
// shared across Nodefall's services. Not every service needs every field;
// services request only the fields relevant to them and get a clear error
// if a required one is missing.
package config

import (
	"errors"
	"os"
)

// Config holds every environment-configured value used across Nodefall.
// Not every service needs every field — auth doesn't care about Redis,
// matchmaker doesn't care about Postgres, etc.
type Config struct {
	JWTSecret string
	RedisURL  string
	DBURL 	  string
	Port 	  string
}

// Load reads all known configuration values from the environment.
// Load itself never fails, since no single service needs every field —
// missing values are just left as empty strings. Callers validate the
// specific fields they require, e.g. RequireJWTSecret below.
func Load() Config {
	return Config{
		JWTSecret: os.Getenv("NODEFALL_JWT_SECRET"),
		RedisURL:  os.Getenv("NODEFALL_REDIS_URL"),
		DBURL:     os.Getenv("NODEFALL_DB_URL"),
		Port:      os.Getenv("NODEFALL_PORT"),
	}
}

// ErrMissingJWTSecret is returned when something that needs signing or
// verification asks for the JWT secret and it isn't set.
var ErrMissingJWTSecret = errors.New("config: NODEFALL_JWT_SECRET not set")

// RequireJWTSecret returns the configured JWT secret as bytes, or
// ErrMissingJWTSecret if it isn't set. This is what shared/jwt will
// call once it's switched over from reading the environment directly.
func (c Config) RequireJWTSecret() ([]byte, error) {
	if c.JWTSecret == "" {
		return nil, ErrMissingJWTSecret
	}
	return []byte(c.JWTSecret), nil
}
