package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUserNotFound is returned when a lookup finds no matching user.
var ErrUserNotFound = errors.New("db: user not found")

// ErrUsernameTaken is returned when registration is attempted with a
// username that's already in use.
var ErrUsernameTaken = errors.New("db: username already taken")

const (
	createUsersTableSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`

	insertUserSQL = `INSERT INTO users (id, username, password_hash) VALUES ($1, $2, $3)`

	selectUserByUsernameSQL = `SELECT id, username, password_hash, created_at FROM users WHERE username = $1`
)

// User represents one registered account. PasswordHash is a bcrypt
// hash, never the plaintext password.
type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

// DB wraps a pooled connection to the auth database.
type DB struct {
	pool *pgxpool.Pool
}

// Connect opens a connection pool to Postgres and verifies it's
// reachable before returning.
func Connect(ctx context.Context, dbURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &DB{pool: pool}, nil
}

// Close releases all pooled connections. Call this on shutdown.
func (db *DB) Close() {
	db.pool.Close()
}

// EnsureSchema creates the users table if it doesn't already exist.
// A simple approach that fits this project's scope — a larger system
// would use a dedicated migration tool instead of an inline CREATE TABLE.
func (db *DB) EnsureSchema(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, createUsersTableSQL)
	return err
}

// CreateUser inserts a new user with the given username and a
// pre-hashed password, returning the generated user ID.
// Returns ErrUsernameTaken if the username is already registered.
func (db *DB) CreateUser(ctx context.Context, username, passwordHash string) (string, error) {
	id := uuid.New().String()

	_, err := db.pool.Exec(ctx, insertUserSQL, id, username, passwordHash)
	if err != nil {
		if isUniqueViolation(err) {
			return "", ErrUsernameTaken
		}
		return "", err
	}
	return id, nil
}

// GetUserByUsername looks up a user by username.
// Returns ErrUserNotFound if no such user exists.
func (db *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := db.pool.QueryRow(ctx, selectUserByUsernameSQL, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505) — used to translate a raw DB error into
// the friendlier ErrUsernameTaken.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
