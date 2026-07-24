// Package security defines opaque bearer-token primitives and credential policy.
// Password hashing and high-entropy token-secret digesting are intentionally
// performed by PostgreSQL, not by this package.
package security

// MaxPasswordLen matches pgcrypto's Blowfish crypt() password limit.
const MaxPasswordLen = 72
