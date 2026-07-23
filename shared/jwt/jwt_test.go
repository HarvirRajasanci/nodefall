package jwt

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestMain(m *testing.M) {
	// Tests run with a fixed secret so they don't depend on the
	// environment they happen to run in.
	os.Setenv("NODEFALL_JWT_SECRET", "test-secret-do-not-use-in-prod")
	m.Run()
}

func TestSignAndVerify_RoundTrip(t *testing.T) {
	token, err := Sign("player-123")
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}

	userID, err := Verify(token)
	if err != nil {
		t.Fatalf("Verify returned error on a freshly signed token: %v", err)
	}
	if userID != "player-123" {
		t.Errorf("got user ID %q, want %q", userID, "player-123")
	}
}

func TestVerify_RejectsTamperedToken(t *testing.T) {
	token, err := Sign("player-123")
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}

	// Flip a character in the signature to simulate tampering.
	tampered := token[:len(token)-1] + "x"

	if _, err := Verify(tampered); err == nil {
		t.Error("Verify accepted a tampered token, want ErrInvalidToken")
	}
}

func TestVerify_RejectsExpiredToken(t *testing.T) {
	key, err := secret()
	if err != nil {
		t.Fatalf("secret() returned error: %v", err)
	}

	// Build an already-expired token directly, bypassing Sign's TTL,
	// so this test doesn't need to sleep for real.
	now := time.Now()
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID: "player-123",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
		},
	})
	signed, err := expired.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}

	if _, err := Verify(signed); err == nil {
		t.Error("Verify accepted an expired token, want ErrInvalidToken")
	}
}

func TestVerify_RejectsGarbageInput(t *testing.T) {
	if _, err := Verify("not.a.jwt"); err == nil {
		t.Error("Verify accepted a garbage string, want ErrInvalidToken")
	}
}
