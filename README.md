# go-todo

[![Actions Status][action-img]][action-url]
[![PkgGoDev][pkg-go-dev-img]][pkg-go-dev-url]

[action-img]: https://github.com/bcomnes/go-todo/actions/workflows/test.yml/badge.svg
[action-url]: https://github.com/bcomnes/go-todo/actions/workflows/test.yml
[pkg-go-dev-img]: https://pkg.go.dev/badge/github.com/bcomnes/go-todo
[pkg-go-dev-url]: https://pkg.go.dev/github.com/bcomnes/go-todo

A standard-library-focused Go todo application with PostgreSQL persistence, server-rendered pages, HTMX interactions, and a parallel JSON API.

## Architecture

HTTP code is organized by feature under `pkg/routes` rather than split into page and API trees.
For example, `pkg/routes/login` owns `GET /login`, `POST /login`, and `POST /api/login`, while `pkg/routes/todos` owns both the todo page actions and `/api/todos` endpoints.
Page templates and their directly renderable HTMX fragments live beside the handlers that render them.

The Go server embeds generated frontend files from `pkg/web/dist`.
TypeScript and CSS are compiled ahead of time by esbuild; the server never invokes a JavaScript toolchain at runtime.
`pkg/web/global.client.ts` installs `htmx.org`, and `pkg/web/global.css` layers application styles over `mine.css`.

## Requirements

- Go 1.26 or newer
- Node.js 22 or newer and npm 10 or newer
- PostgreSQL 17 (earlier supported PostgreSQL versions may also work)

## Development

Create a database, copy the local configuration, install dependencies, and apply migrations:

```console
createdb go_todo
cp .env.example .env
make deps
make migrate-up
```

Build the frontend and start the development server:

```console
make dev
```

`make dev` runs the project-pinned Air tool. Changes to Go files or embedded HTML rebuild and restart the server; changes to `pkg/web/global.client.ts` or `pkg/web/global.css` rebuild the esbuild assets first and then restart Go so the new embedded assets are served.

Open <http://127.0.0.1:8080>.
For local plain HTTP, keep `SESSION_COOKIE_SECURE=false`; deployments served over HTTPS should set it to `true` or omit it to use the secure default.

Useful commands:

```console
make web-build     # type-check TypeScript and build embedded assets
make build         # build assets and all Go packages
make test          # build assets and run all Go tests
make validate      # install/verify dependencies, build, and test
make migrate-list  # show the current schema version
make migrate-down  # roll back one migration
```

Migrations are stored in the top-level `migrations` directory and run with Gostgrator's PostgreSQL command.

## Browser routes

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/` | Landing page and service-status fragment |
| `GET`, `POST` | `/register` | Registration page and form action |
| `GET`, `POST` | `/login` | Login page and form action |
| `POST` | `/logout` | Revoke the browser session |
| `GET` | `/account` | Authenticated account page and fragment |
| `GET`, `POST` | `/todos` | Authenticated todo page and create action |
| `POST` | `/todos/{id}` | Edit a todo |
| `POST` | `/todos/{id}/toggle` | Toggle completion |
| `POST` | `/todos/{id}/delete` | Delete a todo |

Forms work without JavaScript by following `303 See Other` redirects.
With HTMX enabled, mutations replace only the colocated `todo-list` or form fragment.

## JSON API

All data endpoints are explicitly prefixed with `/api`:

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/register` | Create an account and return its bearer token |
| `POST` | `/api/login` | Return an opaque bearer token |
| `POST` | `/api/logout` | Revoke the current bearer token |
| `GET` | `/api/account` | Return the authenticated user |
| `GET`, `POST` | `/api/todos` | List or create todos |
| `GET`, `PATCH`, `DELETE` | `/api/todos/{id}` | Read, update, or delete one owned todo |

Send API credentials as `Authorization: Bearer <token>`.

## Authentication security

Passwords are hashed and verified inside PostgreSQL with `pgcrypto`; plaintext passwords are never stored.
Successful registration creates the account and its first session atomically, so browser users enter `/todos` immediately and API clients receive a bearer token in the registration response.
Unknown-account login attempts still perform dummy Blowfish work to reduce account-existence timing differences.
Expensive password-hashing operations are bounded in-process so they cannot consume the entire database pool.

A login token has the form `gtd_<selector>.<secret>`.
Only the non-secret selector and a SHA-256 digest of the random 256-bit secret are stored, allowing indexed lookup without retaining reusable token material, scanning every token digest, or spending password-hashing capacity on every authenticated request.
Browser tokens are held in `HttpOnly`, `SameSite=Lax` cookies, with `Secure` enabled by default, and cookie-authenticated mutations require a same-origin `Origin` or `Referer`.
Logout revokes the database record before clearing the cookie.

## Tests

Run the complete suite with:

```console
make test
```

Database-backed route tests run when `DATABASE_URL` is set.
Their package-level `TestMain` clones the migrated source database, points the test process at the disposable clone, and force-drops it after the package finishes so test fixtures do not accumulate.
The source database must have no active sessions while PostgreSQL uses it as a clone template.
When `DATABASE_URL` is unset, database-backed tests skip while unit tests still run.

## License

MIT
