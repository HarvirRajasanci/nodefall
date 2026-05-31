package main

import (
	"fmt"
	"log"
	"net/http"
)

func handleRegister(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "register called")
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "login called")
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	fmt.Fprintf(w, "stats called for user %s", userID)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /register", handleRegister)
	mux.HandleFunc("POST /login", handleLogin)
	mux.HandleFunc("GET /stats/{userID}", handleStats)

	fmt.Println("Auth service running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
