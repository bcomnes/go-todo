package security_test

import (
	"strings"
	"testing"

	"github.com/bcomnes/go-todo/pkg/security"
)

func TestTokenGenerationAndParsing(t *testing.T) {
	token, err := security.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if !strings.HasPrefix(token.Plaintext, "gtd_") {
		t.Fatalf("unexpected token prefix: %q", token.Plaintext)
	}

	parsed, err := security.ParseToken(token.Plaintext)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if parsed.ID != token.ID || parsed.Secret != token.Secret {
		t.Fatal("parsed token did not match generated token")
	}

}

func TestParseTokenRejectsMalformedValues(t *testing.T) {
	for _, value := range []string{"", "token", "gtd_", "gtd_a.b", "Bearer gtd_a.b"} {
		if _, err := security.ParseToken(value); err == nil {
			t.Fatalf("ParseToken(%q) unexpectedly succeeded", value)
		}
	}
}
