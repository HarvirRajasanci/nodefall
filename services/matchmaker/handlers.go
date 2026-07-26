package main

import (
	"encoding/json"
	"net/http"

	"nodefall/shared/middleware"
)

type handlers struct {
	queue *Queue
}

type queueStatusResponse struct {
	Status     string `json:"status"`
	MatchID    string `json:"match_id,omitempty"`
	ServerAddr string `json:"server_addr,omitempty"`
}

// handleJoinQueue adds the caller to the matchmaking queue.
func (h *handlers) handleJoinQueue(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "missing verified identity", http.StatusUnauthorized)
		return
	}

	h.queue.Join(playerID)
	w.WriteHeader(http.StatusAccepted)
}

// handleQueueStatus reports whether the caller is still waiting, has
// been matched, or was never queued.
func (h *handlers) handleQueueStatus(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "missing verified identity", http.StatusUnauthorized)
		return
	}

	status, entry := h.queue.Status(playerID)

	resp := queueStatusResponse{Status: status}
	if entry != nil {
		resp.MatchID = entry.MatchID
		resp.ServerAddr = entry.ServerAddr
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleLeaveQueue removes the caller from the matchmaking queue.
func (h *handlers) handleLeaveQueue(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "missing verified identity", http.StatusUnauthorized)
		return
	}

	h.queue.Leave(playerID)
	w.WriteHeader(http.StatusOK)
}
