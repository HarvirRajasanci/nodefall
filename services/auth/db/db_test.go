package db

import (
	"context"
	"os"
	"testing"
	"time"
)

// testDB connects to the database specified by NODEFALL_DB_URL, skipping
// the test entirely if it isn't set — these are integration tests that
// need a real Postgres instance, not something to run in every environment.
func testDB(t *testing.T) *DB {
	t.Helper()

	url := os.Getenv("NODEFALL_DB_URL")
	if url == "" {
		t.Skip("NODEFALL_DB_URL not set, skipping integration test")
	}

	database, err := Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(database.Close)

	if err := database.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema failed: %v", err)
	}

	return database
}

func TestCreateUser_ThenGetUserByUsername(t *testing.T) {
	database := testDB(t)
	ctx := context.Background()

	username := uniqueUsername(t)
	id, err := database.CreateUser(ctx, username, "some-hash")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if id == "" {
		t.Error("got empty ID from CreateUser")
	}

	user, err := database.GetUserByUsername(ctx, username)
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	if user.ID != id {
		t.Errorf("got ID %q, want %q", user.ID, id)
	}
	if user.PasswordHash != "some-hash" {
		t.Errorf("got PasswordHash %q, want %q", user.PasswordHash, "some-hash")
	}
}

func TestCreateUser_RejectsDuplicateUsername(t *testing.T) {
	database := testDB(t)
	ctx := context.Background()

	username := uniqueUsername(t)
	if _, err := database.CreateUser(ctx, username, "hash-one"); err != nil {
		t.Fatalf("first CreateUser failed: %v", err)
	}

	_, err := database.CreateUser(ctx, username, "hash-two")
	if err != ErrUsernameTaken {
		t.Errorf("got err %v, want ErrUsernameTaken", err)
	}
}

func TestGetUserByUsername_ReturnsErrUserNotFoundForUnknownUser(t *testing.T) {
	database := testDB(t)
	ctx := context.Background()

	_, err := database.GetUserByUsername(ctx, uniqueUsername(t))
	if err != ErrUserNotFound {
		t.Errorf("got err %v, want ErrUserNotFound", err)
	}
}

// uniqueUsername returns a username scoped to this specific test run,
// so repeated test runs never collide on the UNIQUE constraint.
func uniqueUsername(t *testing.T) string {
	t.Helper()
	return "testuser-" + t.Name() + "-" + randomSuffix()
}

// randomSuffix returns a high-resolution timestamp string, unique
// enough to avoid collisions between test runs.
func randomSuffix() string {
	return time.Now().Format("20060102150405.000000")
}
