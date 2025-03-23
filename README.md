# go-todo
[![Actions Status][action-img]][action-url]
[![PkgGoDev][pkg-go-dev-img]][pkg-go-dev-url]

A standard-library focused Go JSON API server example with PostgreSQL persistence, full CRUD, authentication, and test coverage.

## Install

```console
go get github.com/bcomnes/go-todo
```

## Development

This project requires a local PostgreSQL database.

### Running Tests

Unit tests run by default:

```bash
make test
```

Integration (PostgreSQL) tests are skipped unless explicitly enabled:

```bash
make test TEST_FLAGS='-args -db'
```

Or manually:

```bash
go test ./... -args -db
```

### Migrations

```bash
make migrate-up
make migrate-down
```

### Environment Variables

The server and tests rely on the following environment variables (can be set in a `.env` file):

```env
DATABASE_URL=postgres://postgres@localhost/go-todo?sslmode=disable
PORT=8080
HOST=0.0.0.0
```

## Usage

```go
package main

import (
  "fmt"
  "github.com/bcomnes/go-todo"
)

func main() {
  fmt.Println("hello world")
}
```

See more examples on [PkgGoDev][pkg-go-dev-url].

## API

See API docs on [PkgGoDev][pkg-go-dev-url].

## License

MIT

[action-img]: https://github.com/bcomnes/go-todo/workflows/test/badge.svg
[action-url]: https://github.com/bcomnes/go-todo/actions
[pkg-go-dev-img]: https://pkg.go.dev/badge/github.com/bcomnes/go-todo
[pkg-go-dev-url]: https://pkg.go.dev/github.com/bcomnes/go-todo
