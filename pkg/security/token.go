package security

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const tokenPrefix = "gtd_"

// ErrInvalidToken reports a bearer token that does not match go-todo's format.
var ErrInvalidToken = errors.New("invalid token")

// Token is an opaque credential split into a non-secret lookup selector and an
// authenticating secret. Plaintext is returned to the client once; callers must
// persist only ID and a one-way digest of Secret.
type Token struct {
	Plaintext string
	ID        string
	Secret    string
}

// GenerateToken creates a cryptographically random selector and 256-bit secret.
func GenerateToken() (Token, error) {
	selector, err := randomURLString(16)
	if err != nil {
		return Token{}, fmt.Errorf("generate token selector: %w", err)
	}
	secret, err := randomURLString(32)
	if err != nil {
		return Token{}, fmt.Errorf("generate token secret: %w", err)
	}

	return Token{
		Plaintext: tokenPrefix + selector + "." + secret,
		ID:        selector,
		Secret:    secret,
	}, nil
}

// ParseToken validates and separates a plaintext token without authenticating
// it. Authentication still requires comparing Secret with its stored digest.
func ParseToken(plaintext string) (Token, error) {
	if !strings.HasPrefix(plaintext, tokenPrefix) {
		return Token{}, ErrInvalidToken
	}

	parts := strings.Split(strings.TrimPrefix(plaintext, tokenPrefix), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Token{}, ErrInvalidToken
	}

	selectorBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(selectorBytes) != 16 {
		return Token{}, ErrInvalidToken
	}
	secretBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(secretBytes) != 32 {
		return Token{}, ErrInvalidToken
	}

	return Token{
		Plaintext: plaintext,
		ID:        parts[0],
		Secret:    parts[1],
	}, nil
}

func randomURLString(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
