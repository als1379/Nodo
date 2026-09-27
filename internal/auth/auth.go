package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email is already registered")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Result struct {
	User      User      `json:"user"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Service struct {
	db         *pgxpool.Pool
	sessionTTL time.Duration
}

func NewService(db *pgxpool.Pool, sessionTTL time.Duration) *Service {
	return &Service{db: db, sessionTTL: sessionTTL}
}

func (s *Service) Register(ctx context.Context, email, password string) (Result, error) {
	email = normalizeEmail(email)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Result{}, fmt.Errorf("hash password: %w", err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback(ctx)
	user := User{ID: newUUID(), Email: email}
	err = tx.QueryRow(ctx, `INSERT INTO users(id,email,password_hash) VALUES($1,$2,$3) RETURNING created_at`, user.ID, user.Email, string(hash)).Scan(&user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_unique" {
			return Result{}, ErrEmailTaken
		}
		return Result{}, err
	}
	result, err := createSession(ctx, tx, user, s.sessionTTL)
	if err != nil {
		return Result{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (Result, error) {
	var user User
	var passwordHash string
	err := s.db.QueryRow(ctx, `SELECT id,email,password_hash,created_at FROM users WHERE email=$1`, normalizeEmail(email)).Scan(&user.ID, &user.Email, &passwordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Keep unknown-user logins close to the cost of a normal password check.
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoO5UGew8QSOXhY9Qd7P1dW8hWrX1G9G7e"), []byte(password))
		return Result{}, ErrInvalidCredentials
	}
	if err != nil {
		return Result{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return Result{}, ErrInvalidCredentials
	}
	return createSession(ctx, s.db, user, s.sessionTTL)
}

func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	var user User
	sum := sha256.Sum256([]byte(token))
	err := s.db.QueryRow(ctx, `SELECT u.id,u.email,u.created_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now()`, sum[:]).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidToken
	}
	return user, err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	sum := sha256.Sum256([]byte(token))
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, sum[:])
	return err
}

type dbtx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func createSession(ctx context.Context, db dbtx, user User, ttl time.Duration) (Result, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Result{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	expires := time.Now().UTC().Add(ttl)
	var id string
	err := db.QueryRow(ctx, `INSERT INTO sessions(id,user_id,token_hash,expires_at) VALUES($1,$2,$3,$4) RETURNING id`, newUUID(), user.ID, sum[:], expires).Scan(&id)
	if err != nil {
		return Result{}, err
	}
	return Result{User: user, Token: token, ExpiresAt: expires}, nil
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable")
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
