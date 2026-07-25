package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"nodefall/services/auth/db"
)

// testHandlers wires up real handlers against a real Postgres instance,
// skipping if NODEFALL_DB_URL isn't set — same convention as db_test.go.
func testHandlers(t *testing.T) *handlers {
	t.Helper()

	url := os.Getenv("NODEFALL_DB_URL")
	if url == "" {
		t.Skip("NODEFALL_DB_URL not set, skipping integration test")
	}

	database, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(database.Close)

	if err := database.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema failed: %v", err)
	}

	return &handlers{db: database}
}

func uniqueUsername(t *testing.T) string {
	t.Helper()
	return "handlertest-" + t.Name() + "-" + time.Now().Format("20060102150405.000000")
}

func doJSON(t *testing.T, handler http.HandlerFunc, body any) *httptest.ResponseRecorder {
	t.Helper()

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func TestHandleRegister_CreatesUser(t *testing.T) {
	h := testHandlers(t)
	username := uniqueUsername(t)

	rec := doJSON(t, h.handleRegister, registerRequest{Username: username, Password: "hunter22"})

	if rec.Code != http.StatusCreated {
		t.Errorf("got status %d, want %d — body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestHandleRegister_RejectsDuplicateUsername(t *testing.T) {
	h := testHandlers(t)
	username := uniqueUsername(t)

	doJSON(t, h.handleRegister, registerRequest{Username: username, Password: "hunter22"})
	rec := doJSON(t, h.handleRegister, registerRequest{Username: username, Password: "different1"})

	if rec.Code != http.StatusConflict {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestHandleRegister_RejectsEmptyFields(t *testing.T) {
	h := testHandlers(t)

	rec := doJSON(t, h.handleRegister, registerRequest{Username: "", Password: ""})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleRegister_RejectsShortPassword(t *testing.T) {
	h := testHandlers(t)
	username := uniqueUsername(t)

	rec := doJSON(t, h.handleRegister, registerRequest{Username: username, Password: "short1"})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleLogin_ReturnsTokenOnValidCredentials(t *testing.T) {
	h := testHandlers(t)
	username := uniqueUsername(t)

	doJSON(t, h.handleRegister, registerRequest{Username: username, Password: "hunter22"})

	rec := doJSON(t, h.handleLogin, loginRequest{Username: username, Password: "hunter22"})

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d — body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp loginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Token == "" {
		t.Error("got empty token, want a signed JWT")
	}
}

func TestHandleLogin_RejectsWrongPassword(t *testing.T) {
	h := testHandlers(t)
	username := uniqueUsername(t)

	doJSON(t, h.handleRegister, registerRequest{Username: username, Password: "hunter22"})
	rec := doJSON(t, h.handleLogin, loginRequest{Username: username, Password: "wrongpass1"})

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleLogin_RejectsUnknownUsername(t *testing.T) {
	h := testHandlers(t)

	rec := doJSON(t, h.handleLogin, loginRequest{Username: uniqueUsername(t), Password: "irrelevant1"})

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
