package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"nodefall/services/auth/db"
	"nodefall/shared/middleware"
)

// friendRequestRequest is the expected JSON body for POST /friends/request.
type friendRequestRequest struct {
	Username string `json:"username"`
}

// friendActionRequest is the expected JSON body for POST /friends/accept
// and DELETE /friends/{id} uses the path instead of a body.
type friendActionRequest struct {
	FriendshipID string `json:"friendship_id"`
}

// friendView is the JSON shape returned for one friend or pending
// request entry.
type friendView struct {
	FriendshipID string `json:"friendship_id"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
}

// friendsListResponse is returned by GET /friends.
type friendsListResponse struct {
	Friends []friendView `json:"friends"`
	Pending []friendView `json:"pending"`
}

func toFriendViews(friends []db.Friend) []friendView {
	views := make([]friendView, 0, len(friends))
	for _, f := range friends {
		views = append(views, friendView{
			FriendshipID: f.FriendshipID,
			UserID:       f.UserID,
			Username:     f.Username,
		})
	}
	return views
}

// handleFriendsList returns the caller's accepted friends and incoming
// pending requests.
func (h *handlers) handleFriendsList(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "missing verified identity", http.StatusUnauthorized)
		return
	}

	friends, err := h.db.ListFriends(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	pending, err := h.db.ListPendingRequests(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(friendsListResponse{
		Friends: toFriendViews(friends),
		Pending: toFriendViews(pending),
	})
}

// handleFriendRequest sends a friend request to the given username.
func (h *handlers) handleFriendRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "missing verified identity", http.StatusUnauthorized)
		return
	}

	var req friendRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	_, err := h.db.RequestFriend(r.Context(), userID, req.Username)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrUserNotFound):
			http.Error(w, "user not found", http.StatusNotFound)
		case errors.Is(err, db.ErrCannotFriendSelf):
			http.Error(w, "cannot friend yourself", http.StatusBadRequest)
		case errors.Is(err, db.ErrAlreadyRequested):
			http.Error(w, "a request or friendship already exists", http.StatusConflict)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// handleFriendAccept accepts a pending friend request.
func (h *handlers) handleFriendAccept(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "missing verified identity", http.StatusUnauthorized)
		return
	}

	var req friendActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FriendshipID == "" {
		http.Error(w, "friendship_id is required", http.StatusBadRequest)
		return
	}

	err := h.db.AcceptFriend(r.Context(), userID, req.FriendshipID)
	if err != nil {
		if errors.Is(err, db.ErrFriendRequestNotFound) {
			http.Error(w, "request not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleFriendRemove removes a friend or declines a pending request.
func (h *handlers) handleFriendRemove(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "missing verified identity", http.StatusUnauthorized)
		return
	}

	friendshipID := r.PathValue("id")
	if friendshipID == "" {
		http.Error(w, "friendship id is required", http.StatusBadRequest)
		return
	}

	err := h.db.RemoveFriend(r.Context(), userID, friendshipID)
	if err != nil {
		if errors.Is(err, db.ErrFriendRequestNotFound) {
			http.Error(w, "request not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
