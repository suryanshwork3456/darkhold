package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrEmailTaken = errors.New("email already registered")

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"` // never sent to clients
	CreatedAt    time.Time `json:"createdAt"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// CreateUser inserts the user and an empty wallet in one transaction.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash, name string) (User, error) {
	u := User{ID: uuid.NewString(), Email: email, Name: name, PasswordHash: passwordHash}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, `
		INSERT INTO users (id, email, name, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`,
		u.ID, u.Email, u.Name, u.PasswordHash).Scan(&u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique violation
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO wallets (user_id) VALUES ($1)`, u.ID); err != nil {
		return User{}, err
	}

	if err := tx.Commit(); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, created_at
		FROM users WHERE email = $1`, email))
}

func (s *Store) GetUserByID(ctx context.Context, id string) (User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, created_at
		FROM users WHERE id = $1`, id))
}

func (s *Store) scanUser(row *sql.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)
	return u, err
}