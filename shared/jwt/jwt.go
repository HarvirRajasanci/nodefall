// Package jwt provides signing and verification of player auth tokens.
// The auth service is the only thing that calls Sign. Every other service
// (matchmaker, game) calls Verify locally with the same shared secret —
// no network round-trip back to auth is ever needed.
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"nodefall/shared/config"
)

// tokenTTL is how long a token is valid after signing.
const tokenTTL = 24 * time.Hour

// ErrInvalidToken covers any failure to verify a token: bad signature,
// expired, malformed, or wrong claims shape. Callers don't need to
// distinguish these cases — the token just isn't valid.
var ErrInvalidToken = errors.New("jwt: invalid token")

// claims is the payload embedded in every Nodefall token.
type claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// secret loads the signing key via shared/config.
func secret() ([]byte, error) {
	return config.Load().RequireJWTSecret()
}

// Sign issues a new token for the given user ID, valid for tokenTTL.
// Only the auth service should call this.
func Sign(userID string) (string, error) {
	key, err := secret()
	if err != nil {
		return "", err
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	})

	return token.SignedString(key)
}

// Verify checks a token's signature and expiry, returning the user ID
// it was issued for. Any service can call this locally — auth doesn't
// need to be reachable or even running for verification to work.
func Verify(tokenString string) (string, error) {
	key, err := secret()
	if err != nil {
		return "", err
	}

	parsed, err := jwt.ParseWithClaims(tokenString, &claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return key, nil
	})
	if err != nil || !parsed.Valid {
		return "", ErrInvalidToken
	}

	c, ok := parsed.Claims.(*claims)
	if !ok || c.UserID == "" {
		return "", ErrInvalidToken
	}

	return c.UserID, nil
}
