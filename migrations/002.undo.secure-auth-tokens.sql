BEGIN;

DROP TABLE public.auth_tokens;

CREATE TABLE public.auth_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX auth_tokens_user_id_idx ON public.auth_tokens(user_id);

CREATE TRIGGER update_auth_tokens_updated_at
BEFORE UPDATE ON public.auth_tokens
FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

COMMIT;
