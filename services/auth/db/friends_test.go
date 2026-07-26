package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

// testFriendsDB connects and ensures both schemas exist — friendships
// depends on users via foreign key, so EnsureSchema must run first.
func testFriendsDB(t *testing.T) *DB {
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
	if err := database.EnsureFriendsSchema(context.Background()); err != nil {
		t.Fatalf("EnsureFriendsSchema failed: %v", err)
	}

	return database
}

// createTestUser is a small helper that registers a throwaway user and
// returns their ID, so friends tests don't need real bcrypt hashing —
// any non-empty string works as a "password hash" here since these
// tests never call login.
func createTestUser(t *testing.T, database *DB, username string) string {
	t.Helper()
	id, err := database.CreateUser(context.Background(), username, "fake-hash")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	return id
}

func uniqueName(t *testing.T, suffix string) string {
	t.Helper()
	return "friendtest-" + t.Name() + "-" + suffix + "-" + time.Now().Format("20060102150405.000000")
}

func TestRequestFriend_CreatesPendingRequest(t *testing.T) {
	database := testFriendsDB(t)
	ctx := context.Background()

	aliceName := uniqueName(t, "alice")
	bobName := uniqueName(t, "bob")
	aliceID := createTestUser(t, database, aliceName)
	createTestUser(t, database, bobName)

	friendshipID, err := database.RequestFriend(ctx, aliceID, bobName)
	if err != nil {
		t.Fatalf("RequestFriend failed: %v", err)
	}
	if friendshipID == "" {
		t.Error("got empty friendship ID")
	}
}

func TestRequestFriend_RejectsSelfFriending(t *testing.T) {
	database := testFriendsDB(t)
	ctx := context.Background()

	name := uniqueName(t, "solo")
	id := createTestUser(t, database, name)

	_, err := database.RequestFriend(ctx, id, name)
	if !errors.Is(err, ErrCannotFriendSelf) {
		t.Errorf("got err %v, want ErrCannotFriendSelf", err)
	}
}

func TestRequestFriend_RejectsDuplicateRequest(t *testing.T) {
	database := testFriendsDB(t)
	ctx := context.Background()

	aliceName := uniqueName(t, "alice")
	bobName := uniqueName(t, "bob")
	aliceID := createTestUser(t, database, aliceName)
	createTestUser(t, database, bobName)

	if _, err := database.RequestFriend(ctx, aliceID, bobName); err != nil {
		t.Fatalf("first RequestFriend failed: %v", err)
	}

	_, err := database.RequestFriend(ctx, aliceID, bobName)
	if !errors.Is(err, ErrAlreadyRequested) {
		t.Errorf("got err %v, want ErrAlreadyRequested", err)
	}
}

func TestRequestFriend_RejectsReverseDuplicateRequest(t *testing.T) {
	database := testFriendsDB(t)
	ctx := context.Background()

	aliceName := uniqueName(t, "alice")
	bobName := uniqueName(t, "bob")
	aliceID := createTestUser(t, database, aliceName)
	bobID := createTestUser(t, database, bobName)

	if _, err := database.RequestFriend(ctx, aliceID, bobName); err != nil {
		t.Fatalf("first RequestFriend failed: %v", err)
	}

	// Bob tries to request Alice while Alice's request to Bob is still
	// pending — should also be rejected, not create a second row.
	_, err := database.RequestFriend(ctx, bobID, aliceName)
	if !errors.Is(err, ErrAlreadyRequested) {
		t.Errorf("got err %v, want ErrAlreadyRequested", err)
	}
}

func TestAcceptFriend_MakesBothUsersFriends(t *testing.T) {
	database := testFriendsDB(t)
	ctx := context.Background()

	aliceName := uniqueName(t, "alice")
	bobName := uniqueName(t, "bob")
	aliceID := createTestUser(t, database, aliceName)
	bobID := createTestUser(t, database, bobName)

	friendshipID, err := database.RequestFriend(ctx, aliceID, bobName)
	if err != nil {
		t.Fatalf("RequestFriend failed: %v", err)
	}

	if err := database.AcceptFriend(ctx, bobID, friendshipID); err != nil {
		t.Fatalf("AcceptFriend failed: %v", err)
	}

	aliceFriends, err := database.ListFriends(ctx, aliceID)
	if err != nil {
		t.Fatalf("ListFriends(alice) failed: %v", err)
	}
	if len(aliceFriends) != 1 || aliceFriends[0].Username != bobName {
		t.Errorf("alice's friends = %+v, want one entry for %q", aliceFriends, bobName)
	}

	bobFriends, err := database.ListFriends(ctx, bobID)
	if err != nil {
		t.Fatalf("ListFriends(bob) failed: %v", err)
	}
	if len(bobFriends) != 1 || bobFriends[0].Username != aliceName {
		t.Errorf("bob's friends = %+v, want one entry for %q", bobFriends, aliceName)
	}
}

func TestAcceptFriend_RejectsWhenCalledByRequester(t *testing.T) {
	database := testFriendsDB(t)
	ctx := context.Background()

	aliceName := uniqueName(t, "alice")
	bobName := uniqueName(t, "bob")
	aliceID := createTestUser(t, database, aliceName)
	createTestUser(t, database, bobName)

	friendshipID, err := database.RequestFriend(ctx, aliceID, bobName)
	if err != nil {
		t.Fatalf("RequestFriend failed: %v", err)
	}

	// Alice (the requester, not the addressee) tries to accept her own
	// request — should be rejected.
	err = database.AcceptFriend(ctx, aliceID, friendshipID)
	if !errors.Is(err, ErrFriendRequestNotFound) {
		t.Errorf("got err %v, want ErrFriendRequestNotFound", err)
	}
}

func TestListPendingRequests_ShowsIncomingRequest(t *testing.T) {
	database := testFriendsDB(t)
	ctx := context.Background()

	aliceName := uniqueName(t, "alice")
	bobName := uniqueName(t, "bob")
	aliceID := createTestUser(t, database, aliceName)
	bobID := createTestUser(t, database, bobName)

	if _, err := database.RequestFriend(ctx, aliceID, bobName); err != nil {
		t.Fatalf("RequestFriend failed: %v", err)
	}

	pending, err := database.ListPendingRequests(ctx, bobID)
	if err != nil {
		t.Fatalf("ListPendingRequests failed: %v", err)
	}
	if len(pending) != 1 || pending[0].Username != aliceName {
		t.Errorf("bob's pending requests = %+v, want one entry from %q", pending, aliceName)
	}
}

func TestRemoveFriend_DeletesRelationship(t *testing.T) {
	database := testFriendsDB(t)
	ctx := context.Background()

	aliceName := uniqueName(t, "alice")
	bobName := uniqueName(t, "bob")
	aliceID := createTestUser(t, database, aliceName)
	bobID := createTestUser(t, database, bobName)

	friendshipID, err := database.RequestFriend(ctx, aliceID, bobName)
	if err != nil {
		t.Fatalf("RequestFriend failed: %v", err)
	}
	if err := database.AcceptFriend(ctx, bobID, friendshipID); err != nil {
		t.Fatalf("AcceptFriend failed: %v", err)
	}

	if err := database.RemoveFriend(ctx, aliceID, friendshipID); err != nil {
		t.Fatalf("RemoveFriend failed: %v", err)
	}

	friends, err := database.ListFriends(ctx, aliceID)
	if err != nil {
		t.Fatalf("ListFriends failed: %v", err)
	}
	if len(friends) != 0 {
		t.Errorf("got %d friends after removal, want 0", len(friends))
	}
}

func TestRemoveFriend_RejectsNonParticipant(t *testing.T) {
	database := testFriendsDB(t)
	ctx := context.Background()

	aliceName := uniqueName(t, "alice")
	bobName := uniqueName(t, "bob")
	charlieName := uniqueName(t, "charlie")
	aliceID := createTestUser(t, database, aliceName)
	createTestUser(t, database, bobName)
	charlieID := createTestUser(t, database, charlieName)

	friendshipID, err := database.RequestFriend(ctx, aliceID, bobName)
	if err != nil {
		t.Fatalf("RequestFriend failed: %v", err)
	}

	err = database.RemoveFriend(ctx, charlieID, friendshipID)
	if !errors.Is(err, ErrFriendRequestNotFound) {
		t.Errorf("got err %v, want ErrFriendRequestNotFound", err)
	}
}
