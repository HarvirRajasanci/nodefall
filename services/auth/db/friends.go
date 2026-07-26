package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrCannotFriendSelf is returned when a user tries to friend themselves.
var ErrCannotFriendSelf = errors.New("db: cannot send a friend request to yourself")

// ErrAlreadyRequested is returned when a pending or accepted relationship
// already exists between the two users, in either direction.
var ErrAlreadyRequested = errors.New("db: a friend request or friendship already exists")

// ErrFriendRequestNotFound is returned when a friendship row doesn't
// exist, or exists but the calling user isn't a participant in it —
// deliberately the same error either way, so this can't be used to
// probe whether some ID exists.
var ErrFriendRequestNotFound = errors.New("db: friend request not found")

const (
	createFriendshipsTableSQL = `
		CREATE TABLE IF NOT EXISTS friendships (
			id UUID PRIMARY KEY,
			requester_id UUID NOT NULL REFERENCES users(id),
			addressee_id UUID NOT NULL REFERENCES users(id),
			status TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`

	selectExistingRelationshipSQL = `
		SELECT id FROM friendships
		WHERE (requester_id = $1 AND addressee_id = $2)
		   OR (requester_id = $2 AND addressee_id = $1)
	`

	insertFriendshipSQL = `
		INSERT INTO friendships (id, requester_id, addressee_id, status)
		VALUES ($1, $2, $3, 'pending')
	`

	selectFriendshipByIDSQL = `
		SELECT id, requester_id, addressee_id, status
		FROM friendships WHERE id = $1
	`

	updateFriendshipStatusSQL = `UPDATE friendships SET status = 'accepted' WHERE id = $1`

	deleteFriendshipSQL = `DELETE FROM friendships WHERE id = $1`

	selectFriendsSQL = `
		SELECT f.id,
		       CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END,
		       u.username
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END
		WHERE f.status = 'accepted' AND (f.requester_id = $1 OR f.addressee_id = $1)
	`

	selectPendingRequestsSQL = `
		SELECT f.id, f.requester_id, u.username
		FROM friendships f
		JOIN users u ON u.id = f.requester_id
		WHERE f.addressee_id = $1 AND f.status = 'pending'
	`
)

// Friend represents one entry in a friends or pending-requests list:
// the friendship row's ID, the other user's ID, and their username.
type Friend struct {
	FriendshipID string
	UserID       string
	Username     string
}

// EnsureFriendsSchema creates the friendships table if it doesn't
// already exist. Must be called after EnsureSchema, since friendships
// references users(id) as a foreign key.
func (db *DB) EnsureFriendsSchema(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, createFriendshipsTableSQL)
	return err
}

// RequestFriend creates a pending friend request from requesterID to
// the user with the given username. Returns ErrCannotFriendSelf if
// they match the same user, or ErrAlreadyRequested if any pending or
// accepted relationship already exists between them in either direction.
func (db *DB) RequestFriend(ctx context.Context, requesterID, addresseeUsername string) (string, error) {
	addressee, err := db.GetUserByUsername(ctx, addresseeUsername)
	if err != nil {
		return "", err // propagates ErrUserNotFound as-is
	}

	if addressee.ID == requesterID {
		return "", ErrCannotFriendSelf
	}

	var existing string
	err = db.pool.QueryRow(ctx, selectExistingRelationshipSQL, requesterID, addressee.ID).Scan(&existing)
	if err == nil {
		return "", ErrAlreadyRequested
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	id := uuid.New().String()
	_, err = db.pool.Exec(ctx, insertFriendshipSQL, id, requesterID, addressee.ID)
	if err != nil {
		return "", err
	}
	return id, nil
}

// AcceptFriend marks a pending friend request as accepted. Only the
// addressee (the user who received the request) can accept it.
func (db *DB) AcceptFriend(ctx context.Context, userID, friendshipID string) error {
	var id, requesterID, addresseeID, status string
	err := db.pool.QueryRow(ctx, selectFriendshipByIDSQL, friendshipID).
		Scan(&id, &requesterID, &addresseeID, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrFriendRequestNotFound
		}
		return err
	}

	if addresseeID != userID || status != "pending" {
		return ErrFriendRequestNotFound
	}

	_, err = db.pool.Exec(ctx, updateFriendshipStatusSQL, friendshipID)
	return err
}

// RemoveFriend deletes a friendship row — used both to decline a
// pending request and to remove an existing friend. Either participant
// (requester or addressee) can do this.
func (db *DB) RemoveFriend(ctx context.Context, userID, friendshipID string) error {
	var id, requesterID, addresseeID, status string
	err := db.pool.QueryRow(ctx, selectFriendshipByIDSQL, friendshipID).
		Scan(&id, &requesterID, &addresseeID, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrFriendRequestNotFound
		}
		return err
	}

	if requesterID != userID && addresseeID != userID {
		return ErrFriendRequestNotFound
	}

	_, err = db.pool.Exec(ctx, deleteFriendshipSQL, friendshipID)
	return err
}

// ListFriends returns every accepted friend of userID.
func (db *DB) ListFriends(ctx context.Context, userID string) ([]Friend, error) {
	rows, err := db.pool.Query(ctx, selectFriendsSQL, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []Friend
	for rows.Next() {
		var f Friend
		if err := rows.Scan(&f.FriendshipID, &f.UserID, &f.Username); err != nil {
			return nil, err
		}
		friends = append(friends, f)
	}
	return friends, rows.Err()
}

// ListPendingRequests returns every incoming (not yet accepted) friend
// request addressed to userID.
func (db *DB) ListPendingRequests(ctx context.Context, userID string) ([]Friend, error) {
	rows, err := db.pool.Query(ctx, selectPendingRequestsSQL, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pending []Friend
	for rows.Next() {
		var f Friend
		if err := rows.Scan(&f.FriendshipID, &f.UserID, &f.Username); err != nil {
			return nil, err
		}
		pending = append(pending, f)
	}
	return pending, rows.Err()
}
