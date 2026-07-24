-- Down Migration: drop auth_tokens, todos, users, and triggers

DROP TRIGGER IF EXISTS update_auth_tokens_updated_at ON public.auth_tokens;
DROP TRIGGER IF EXISTS update_todos_updated_at ON public.todos;
DROP TRIGGER IF EXISTS update_users_updated_at ON public.users;

DROP FUNCTION IF EXISTS public.update_updated_at_column();

DROP TABLE IF EXISTS public.auth_tokens;
DROP TABLE IF EXISTS public.todos;
DROP TABLE IF EXISTS public.users;
