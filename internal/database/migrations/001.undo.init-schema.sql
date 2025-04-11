-- Down Migration: drop auth_tokens, todos, users, and triggers

DROP TRIGGER IF EXISTS update_auth_tokens_updated_at ON auth_tokens;
DROP TRIGGER IF EXISTS update_todos_updated_at ON todos;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS auth_tokens;
DROP TABLE IF EXISTS todos;
DROP TABLE IF EXISTS users;
