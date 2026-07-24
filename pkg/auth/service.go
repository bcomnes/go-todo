// Package auth implements account credential, opaque-token, and authenticated
// session operations shared by browser and JSON routes.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bcomnes/go-todo/pkg/models"
	"github.com/bcomnes/go-todo/pkg/security"
	"github.com/jackc/pgx/v5/pgconn"
)

const defaultHashConcurrency = 4

var (
	// ErrCapacity reports that this process has admitted its maximum number of
	// concurrent pgcrypto password-hashing operations.
	ErrCapacity = errors.New("password hashing capacity is full")
	// ErrUserExists reports a registration conflict without revealing which
	// unique account field already exists.
	ErrUserExists = errors.New("username or email already exists")
	// ErrInvalidCredentials reports a failed login without distinguishing an
	// unknown account from a wrong password.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUnavailable reports that credential verification could not be completed.
	ErrUnavailable = errors.New("authentication unavailable")
	// ErrTokenCreation reports that a successfully authenticated login could not
	// issue or persist its opaque bearer token.
	ErrTokenCreation = errors.New("failed to create token")
	// ErrUnauthorized reports a missing, malformed, expired, revoked, or
	// incorrectly authenticated bearer token.
	ErrUnauthorized = errors.New("unauthorized")
)

// Options controls resource limits for expensive password operations.
type Options struct {
	HashConcurrency int
}

// Registration contains normalized account-creation inputs.
type Registration struct {
	Username string
	Email    string
	Password string
}

// Credentials contains login inputs.
type Credentials struct {
	Email    string
	Password string
}

// AuthResult contains the public user and the one-time plaintext bearer token
// returned after either registration or login.
type AuthResult struct {
	Token     string      `json:"token"`
	TokenType string      `json:"token_type"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      models.User `json:"user"`
}

// Session is the authenticated user and non-secret token selector attached to a
// request. TokenID is sufficient to revoke the current token but not reuse it.
type Session struct {
	User    models.User
	TokenID string
}

// Service owns the shared PostgreSQL credential operations and bounds expensive
// password hashing before it can consume the entire connection pool.
type Service struct {
	db        *sql.DB
	tokenTTL  time.Duration
	hashSlots chan struct{}
}

// New creates an authentication service backed by a migrated PostgreSQL database.
func New(db *sql.DB, tokenTTL time.Duration, options Options) (*Service, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	if tokenTTL <= 0 {
		return nil, errors.New("token TTL must be greater than zero")
	}

	hashConcurrency := options.HashConcurrency
	if hashConcurrency <= 0 {
		hashConcurrency = defaultHashConcurrency
	}
	return &Service{
		db:        db,
		tokenTTL:  tokenTTL,
		hashSlots: make(chan struct{}, hashConcurrency),
	}, nil
}

// Register atomically creates an account and its first authenticated session.
// PostgreSQL hashes the password; only a digest of the issued token secret is
// persisted. A session-storage or commit failure rolls back the new account.
func (service *Service) Register(ctx context.Context, registration Registration) (AuthResult, error) {
	if err := service.acquireHashSlot(ctx); err != nil {
		return AuthResult{}, fmt.Errorf("wait for password hashing: %w", err)
	}
	defer service.releaseHashSlot()

	registration.Username = strings.TrimSpace(registration.Username)
	registration.Email = strings.ToLower(strings.TrimSpace(registration.Email))
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return AuthResult{}, fmt.Errorf("begin registration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var user models.User
	err = tx.QueryRowContext(ctx, `
		INSERT INTO public.users (username, email, password_hash)
		VALUES ($1, $2, public.crypt($3, public.gen_salt('bf', 12)))
		RETURNING id, username, email, created_at, updated_at
	`, registration.Username, registration.Email, registration.Password).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return AuthResult{}, ErrUserExists
		}
		return AuthResult{}, fmt.Errorf("insert user: %w", err)
	}

	token, expiresAt, err := service.issueToken(ctx, tx, user.ID)
	if err != nil {
		return AuthResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return AuthResult{}, fmt.Errorf("commit registration: %w", err)
	}
	return newAuthResult(user, token, expiresAt), nil
}

// Login verifies a password with PostgreSQL and persists only a digest of the
// new high-entropy token secret. The plaintext token is returned once.
func (service *Service) Login(ctx context.Context, credentials Credentials) (AuthResult, error) {
	credentials.Email = strings.ToLower(strings.TrimSpace(credentials.Email))
	if credentials.Email == "" || credentials.Password == "" || len(credentials.Password) > security.MaxPasswordLen {
		return AuthResult{}, ErrInvalidCredentials
	}
	if err := service.acquireHashSlot(ctx); err != nil {
		return AuthResult{}, fmt.Errorf("%w: wait for password verification: %w", ErrUnavailable, err)
	}
	defer service.releaseHashSlot()

	var (
		id              sql.NullInt64
		username        sql.NullString
		email           sql.NullString
		createdAt       sql.NullTime
		updatedAt       sql.NullTime
		passwordMatches sql.NullBool
	)
	err := service.db.QueryRowContext(ctx, `
		WITH candidate AS MATERIALIZED (
			SELECT id, username, email, password_hash, created_at, updated_at
			FROM public.users
			WHERE email = $1
		),
		checked AS MATERIALIZED (
			SELECT public.crypt(
				$2,
				COALESCE(
					(SELECT password_hash FROM candidate),
					public.gen_salt('bf', 12)
				)
			) AS password_hash
		)
		SELECT
			candidate.id,
			candidate.username,
			candidate.email,
			candidate.created_at,
			candidate.updated_at,
			candidate.password_hash = checked.password_hash
		FROM checked
		LEFT JOIN candidate ON TRUE
	`, credentials.Email, credentials.Password).Scan(
		&id,
		&username,
		&email,
		&createdAt,
		&updatedAt,
		&passwordMatches,
	)
	if err != nil {
		return AuthResult{}, fmt.Errorf("%w: query credentials: %v", ErrUnavailable, err)
	}
	if !id.Valid || !username.Valid || !email.Valid || !createdAt.Valid || !updatedAt.Valid ||
		!passwordMatches.Valid || !passwordMatches.Bool {
		return AuthResult{}, ErrInvalidCredentials
	}
	user := models.User{
		ID:        id.Int64,
		Username:  username.String,
		Email:     email.String,
		CreatedAt: createdAt.Time,
		UpdatedAt: updatedAt.Time,
	}

	token, expiresAt, err := service.issueToken(ctx, service.db, user.ID)
	if err != nil {
		return AuthResult{}, err
	}
	return newAuthResult(user, token, expiresAt), nil
}

type tokenStore interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func (service *Service) issueToken(ctx context.Context, store tokenStore, userID int64) (security.Token, time.Time, error) {
	token, err := security.GenerateToken()
	if err != nil {
		return security.Token{}, time.Time{}, fmt.Errorf("%w: generate token: %v", ErrTokenCreation, err)
	}
	expiresAt := time.Now().UTC().Add(service.tokenTTL)
	_, err = store.ExecContext(ctx, `
		INSERT INTO public.auth_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, public.digest($3, 'sha256'), $4)
	`, token.ID, userID, token.Secret, expiresAt)
	if err != nil {
		return security.Token{}, time.Time{}, fmt.Errorf("%w: store token: %v", ErrTokenCreation, err)
	}
	return token, expiresAt, nil
}

func newAuthResult(user models.User, token security.Token, expiresAt time.Time) AuthResult {
	return AuthResult{
		Token:     token.Plaintext,
		TokenType: "Bearer",
		ExpiresAt: expiresAt,
		User:      user,
	}
}

// Authenticate verifies a plaintext opaque token against its PostgreSQL digest.
func (service *Service) Authenticate(ctx context.Context, plaintext string) (Session, error) {
	token, err := security.ParseToken(plaintext)
	if err != nil {
		return Session{}, ErrUnauthorized
	}

	var user models.User
	err = service.db.QueryRowContext(ctx, `
		SELECT
			users.id,
			users.username,
			users.email,
			users.created_at,
			users.updated_at
		FROM public.auth_tokens AS auth_tokens
		JOIN public.users AS users ON users.id = auth_tokens.user_id
		WHERE auth_tokens.id = $1
		  AND auth_tokens.token_hash = public.digest($2, 'sha256')
		  AND auth_tokens.revoked_at IS NULL
		  AND auth_tokens.expires_at > CURRENT_TIMESTAMP
	`, token.ID, token.Secret).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrUnauthorized
	}
	if err != nil {
		return Session{}, fmt.Errorf("%w: authenticate token: %v", ErrUnavailable, err)
	}
	return Session{User: user, TokenID: token.ID}, nil
}

// Revoke marks the token represented by session as no longer authenticating.
func (service *Service) Revoke(ctx context.Context, session Session) error {
	result, err := service.db.ExecContext(ctx, `
		UPDATE public.auth_tokens
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`, session.TokenID, session.User.ID)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read revoked token count: %w", err)
	}
	if rows != 1 {
		return ErrUnauthorized
	}
	return nil
}

func (service *Service) acquireHashSlot(ctx context.Context) error {
	select {
	case service.hashSlots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrCapacity
	}
}

func (service *Service) releaseHashSlot() {
	<-service.hashSlots
}
