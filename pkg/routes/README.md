# Route packages

This tree contains go-todo's HTTP endpoints, organized by feature rather than by response format.

A feature package owns all of its transport-facing pieces:

- browser page handlers;
- JSON API handlers;
- route registration;
- request preparation and feature-specific validation;
- page data;
- embedded HTML templates and HTMX fragments; and
- focused route tests.

For example, `login` registers `GET /login`, `POST /login`, and `POST /api/login`, while `todos` owns the todo page, HTMX actions, and `/api/todos` CRUD endpoints. Keeping each feature together avoids maintaining parallel browser-page and API trees.

## Conventions

Each feature package has a `routes.go` file with an explicit `Register` function. Larger features split handlers by method and surface, such as `get-page.go`, `post-page.go`, and `post-api.go`.

The root `routes.go` is the complete route manifest. Go route packages are registered explicitly rather than discovered from the filesystem, so missing registrations fail review visibly and imports remain compile-time checked.

Page-owning packages keep `page.go` and `page.gohtml` next to their handlers. A page template contains both its full-page `content` definition and any directly renderable HTMX fragments.

Cross-feature integration tests live at the root of this tree. Tests concerning only one feature should live in that feature package.

## Shared boundaries

Route packages may depend on these lower-level packages:

- `pkg/auth` for account credentials, token verification, and revocation;
- `pkg/httpx` for bounded request decoding, sessions, same-origin checks, responses, HTMX redirects, and rendering adapters;
- `pkg/web` for the generic page renderer and embedded browser assets; and
- domain packages for feature-specific application operations.

Route packages must not import `pkg/httpapi`. The `httpapi` package imports this tree as the public composition facade, so importing it back would create a cycle.

Do not move feature policy into `pkg/httpx`. Shared transport mechanics belong there; validation rules and response decisions that only one feature needs stay with that feature.
