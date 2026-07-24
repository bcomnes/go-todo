// Package testdatabase creates disposable PostgreSQL clones for integration tests.
//
// A clone inherits the source database's migrated schema and seed state, but all
// fixtures written by a package's tests are isolated and discarded afterward.
// PostgreSQL requires the source database to have no active sessions while it is
// used as a CREATE DATABASE template.
package testdatabase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/bcomnes/go-todo/pkg/database"
)

type runner interface {
	Run() int
}

// Run clones the database named by DATABASE_URL, points the current test process
// at the clone, runs tests, and force-drops the clone even when tests fail. When
// DATABASE_URL is unset, tests run unchanged and may use their normal skip
// behavior. Run is intended to be called from a package's TestMain.
func Run(tests runner) int {
	sourceURL := os.Getenv("DATABASE_URL")
	if sourceURL == "" {
		return tests.Run()
	}

	cloneURL, cleanup, err := clone(sourceURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create disposable test database: %v\n", err)
		return 1
	}
	if err := os.Setenv("DATABASE_URL", cloneURL); err != nil {
		_ = cleanup()
		fmt.Fprintf(os.Stderr, "configure disposable test database: %v\n", err)
		return 1
	}

	code := tests.Run()
	if err := cleanup(); err != nil {
		fmt.Fprintf(os.Stderr, "drop disposable test database: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	return code
}

func clone(sourceURL string) (cloneURL string, cleanup func() error, err error) {
	parsed, err := url.Parse(sourceURL)
	if err != nil {
		return "", nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	sourceDatabase := strings.TrimPrefix(parsed.Path, "/")
	if sourceDatabase == "" || strings.Contains(sourceDatabase, "/") {
		return "", nil, errors.New("DATABASE_URL must identify one database")
	}

	var random [6]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", nil, fmt.Errorf("generate clone name: %w", err)
	}
	cloneDatabase := fmt.Sprintf("go_todo_test_%d_%s", os.Getpid(), hex.EncodeToString(random[:]))

	maintenanceURL := *parsed
	maintenanceDatabase := "postgres"
	if sourceDatabase == maintenanceDatabase {
		maintenanceDatabase = "template1"
	}
	maintenanceURL.Path = "/" + maintenanceDatabase
	maintenanceURL.RawPath = ""
	maintenance, err := database.Connect(context.Background(), maintenanceURL.String())
	if err != nil {
		return "", nil, fmt.Errorf("connect maintenance database: %w", err)
	}

	createSQL := "CREATE DATABASE " + quoteIdentifier(cloneDatabase) + " WITH TEMPLATE " + quoteIdentifier(sourceDatabase)
	if _, err := maintenance.Exec(createSQL); err != nil {
		_ = maintenance.Close()
		return "", nil, fmt.Errorf("clone %s (ensure it has no active sessions): %w", sourceDatabase, err)
	}

	clonedURL := *parsed
	clonedURL.Path = "/" + cloneDatabase
	clonedURL.RawPath = ""
	cleanup = func() error {
		_, dropErr := maintenance.Exec("DROP DATABASE IF EXISTS " + quoteIdentifier(cloneDatabase) + " WITH (FORCE)")
		closeErr := maintenance.Close()
		return errors.Join(dropErr, closeErr)
	}
	return clonedURL.String(), cleanup, nil
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
