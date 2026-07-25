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

The root `routes.go` is the complete route manifest. Go route packages are registered explicitly rather than discovered from the filesystem, so missing registrations fail review visibly and imports remain compile-time checked. It also creates the shared Huma API and authenticated operation group; feature packages register typed JSON operations with Huma while registering browser handlers directly on `http.ServeMux`.

Page-owning packages keep `page.go` near their handlers. A small feature may colocate one `page.gohtml`; a larger feature may create asset/template-only page directories without creating Go subpackages, as `todos/{index,detail,edit}` does. Each template contains its full-page `content` definition and only the HTMX fragments it directly renders. Optional colocated `page.client.ts` files become stable directory-based entries, and relative imports shared by multiple entries are factored into ESM chunks. JSON operation files define Huma input/output structs and explicit operation metadata so request validation, response typing, JSON Schema, and OpenAPI stay synchronized.

Cross-feature integration tests live at the root of this tree. Tests concerning only one feature should live in that feature package.

## Shared boundaries

Route packages may depend on these lower-level packages:

- `pkg/auth` for account credentials, token verification, and revocation;
- `pkg/httpx` for bounded request decoding, sessions, same-origin checks, responses, HTMX redirects, and rendering adapters;
- `pkg/web` for the generic page renderer and embedded browser assets; and
- domain packages for feature-specific application operations.

Route packages must not import `pkg/httpapi`. The `httpapi` package imports this tree as the public composition facade, so importing it back would create a cycle.

Do not move feature policy into `pkg/httpx`. Shared transport mechanics belong there; validation rules and response decisions that only one feature needs stay with that feature.
