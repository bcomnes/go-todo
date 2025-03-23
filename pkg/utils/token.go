package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func GenerateToken() (string, error) {
	b := make([]byte, 32) // 256-bit token
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
