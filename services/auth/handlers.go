package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"nodefall/services/auth/db"
	"nodefall/shared/jwt"
)

const minPasswordLength = 8

// registerRequest is the expected JSON body for POST /register.
type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginRequest is the expected JSON body for POST /login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginResponse is returned on successful login.
type loginResponse struct {
	Token string `json:"token"`
}

// handlers holds the dependencies HTTP handlers need. A struct instead
// of package-level functions so main.go can construct it once with a
// real *db.DB and wire it into the mux.
type handlers struct {
	db *db.DB
}

// handleRegister creates a new account. On success, responds 201 with
// no body; the client is expected to log in separately afterward.
func (h *handlers) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < minPasswordLength {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	_, err = h.db.CreateUser(r.Context(), req.Username, string(hash))
	if err != nil {
		if errors.Is(err, db.ErrUsernameTaken) {
			http.Error(w, "username already taken", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// handleLogin verifies credentials and, on success, responds with a
// signed JWT the client uses for every subsequent request.
func (h *handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.db.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		// Same response whether the username doesn't exist or the
		// password is wrong — don't reveal which one it was.
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := jwt.Sign(user.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{Token: token})
}
