BEGIN;

-- Existing tokens cannot be converted to selector/secret credentials because the
-- selector was never stored separately. Authentication sessions are ephemeral,
-- so invalidate them while preserving users and todos.
DROP TRIGGER IF EXISTS update_auth_tokens_updated_at ON public.auth_tokens;
DROP TABLE public.auth_tokens;

-- The ID is a random, non-secret selector used for indexed lookup. The
-- authenticating secret is returned once and never persisted. Because the secret
-- has 256 bits of entropy, a fast SHA-256 digest is appropriate and avoids using
-- password-hashing capacity on every authenticated request.
CREATE TABLE public.auth_tokens (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (length(id) BETWEEN 20 AND 64),
    CHECK (octet_length(token_hash) = 32),
    CHECK (expires_at > created_at),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE INDEX auth_tokens_user_id_idx ON public.auth_tokens(user_id);
CREATE INDEX auth_tokens_active_expiry_idx
    ON public.auth_tokens(expires_at)
    WHERE revoked_at IS NULL;

COMMIT;
