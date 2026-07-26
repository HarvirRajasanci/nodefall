package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nodefall/shared/jwt"
	"nodefall/shared/middleware"
)

// authedRequest builds an httptest.NewRecorder + *http.Request pair for
// the given handler, wrapped in the real middleware.WithAuth using a
// genuine signed JWT for userID — exercising the actual auth path, not
// a hand-built context.
func authedRequest(t *testing.T, method, path, userID string, body any) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()

	token, err := jwt.Sign(userID)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	var reqBody *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path+"?token="+token, reqBody)
	rec := httptest.NewRecorder()
	return rec, req
}

// callAuthed wraps handler in middleware.WithAuth and serves the request.
func callAuthed(t *testing.T, handler http.HandlerFunc, method, path, userID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	rec, req := authedRequest(t, method, path, userID, body)
	middleware.WithAuth(handler).ServeHTTP(rec, req)
	return rec
}

// registerUser is a small helper: registers a fresh user via the real
// handler and returns their ID.
func registerUser(t *testing.T, h *handlers, username string) string {
	t.Helper()

	rec := doJSON(t, h.handleRegister, registerRequest{Username: username, Password: "hunter22"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register failed: status %d, body %s", rec.Code, rec.Body.String())
	}

	user, err := h.db.GetUserByUsername(t.Context(), username)
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	return user.ID
}

func TestHandleFriendRequest_CreatesRequest(t *testing.T) {
	h := testHandlers(t)
	aliceID := registerUser(t, h, uniqueUsername(t)+"-alice")
	bobName := uniqueUsername(t) + "-bob"
	registerUser(t, h, bobName)

	rec := callAuthed(t, h.handleFriendRequest, http.MethodPost, "/friends/request", aliceID,
		friendRequestRequest{Username: bobName})

	if rec.Code != http.StatusCreated {
		t.Errorf("got status %d, want %d — body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestHandleFriendRequest_RejectsUnknownUsername(t *testing.T) {
	h := testHandlers(t)
	aliceID := registerUser(t, h, uniqueUsername(t)+"-alice")

	rec := callAuthed(t, h.handleFriendRequest, http.MethodPost, "/friends/request", aliceID,
		friendRequestRequest{Username: "nonexistent-user-xyz"})

	if rec.Code != http.StatusNotFound {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleFriendRequest_RejectsSelfFriending(t *testing.T) {
	h := testHandlers(t)
	name := uniqueUsername(t) + "-solo"
	id := registerUser(t, h, name)

	rec := callAuthed(t, h.handleFriendRequest, http.MethodPost, "/friends/request", id,
		friendRequestRequest{Username: name})

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestFriendsFlow_RequestAcceptAndList(t *testing.T) {
	h := testHandlers(t)
	bobName := uniqueUsername(t) + "-bob"
	aliceID := registerUser(t, h, uniqueUsername(t)+"-alice")
	bobID := registerUser(t, h, bobName)

	reqRec := callAuthed(t, h.handleFriendRequest, http.MethodPost, "/friends/request", aliceID,
		friendRequestRequest{Username: bobName})
	if reqRec.Code != http.StatusCreated {
		t.Fatalf("request failed: status %d, body %s", reqRec.Code, reqRec.Body.String())
	}

	// Bob checks his pending requests.
	listRec := callAuthed(t, h.handleFriendsList, http.MethodGet, "/friends", bobID, nil)
	var listResp friendsListResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to decode friends list: %v", err)
	}
	if len(listResp.Pending) != 1 {
		t.Fatalf("got %d pending requests, want 1", len(listResp.Pending))
	}
	friendshipID := listResp.Pending[0].FriendshipID

	// Bob accepts.
	acceptRec := callAuthed(t, h.handleFriendAccept, http.MethodPost, "/friends/accept", bobID,
		friendActionRequest{FriendshipID: friendshipID})
	if acceptRec.Code != http.StatusOK {
		t.Fatalf("accept failed: status %d, body %s", acceptRec.Code, acceptRec.Body.String())
	}

	// Alice should now see Bob as a friend.
	aliceListRec := callAuthed(t, h.handleFriendsList, http.MethodGet, "/friends", aliceID, nil)
	var aliceListResp friendsListResponse
	if err := json.Unmarshal(aliceListRec.Body.Bytes(), &aliceListResp); err != nil {
		t.Fatalf("failed to decode alice's friends list: %v", err)
	}
	if len(aliceListResp.Friends) != 1 || aliceListResp.Friends[0].UserID != bobID {
		t.Errorf("alice's friends = %+v, want one entry for bob (%s)", aliceListResp.Friends, bobID)
	}
}

func TestHandleFriendRemove_DeletesFriendship(t *testing.T) {
	h := testHandlers(t)
	bobName := uniqueUsername(t) + "-bob"
	aliceID := registerUser(t, h, uniqueUsername(t)+"-alice")
	bobID := registerUser(t, h, bobName)

	reqRec := callAuthed(t, h.handleFriendRequest, http.MethodPost, "/friends/request", aliceID,
		friendRequestRequest{Username: bobName})
	if reqRec.Code != http.StatusCreated {
		t.Fatalf("request failed: %s", reqRec.Body.String())
	}

	listRec := callAuthed(t, h.handleFriendsList, http.MethodGet, "/friends", bobID, nil)
	var listResp friendsListResponse
	json.Unmarshal(listRec.Body.Bytes(), &listResp)
	friendshipID := listResp.Pending[0].FriendshipID

	acceptRec := callAuthed(t, h.handleFriendAccept, http.MethodPost, "/friends/accept", bobID,
		friendActionRequest{FriendshipID: friendshipID})
	if acceptRec.Code != http.StatusOK {
		t.Fatalf("accept failed: %s", acceptRec.Body.String())
	}

	// Use a mux so PathValue("id") resolves correctly for the DELETE route.
	mux := http.NewServeMux()
	mux.Handle("DELETE /friends/{id}", middleware.WithAuth(http.HandlerFunc(h.handleFriendRemove)))

	token, err := jwt.Sign(aliceID)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodDelete, "/friends/"+friendshipID+"?token="+token, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("remove failed: status %d, body %s", rec.Code, rec.Body.String())
	}

	aliceListRec := callAuthed(t, h.handleFriendsList, http.MethodGet, "/friends", aliceID, nil)
	var aliceListResp friendsListResponse
	json.Unmarshal(aliceListRec.Body.Bytes(), &aliceListResp)
	if len(aliceListResp.Friends) != 0 {
		t.Errorf("got %d friends after removal, want 0", len(aliceListResp.Friends))
	}
}
