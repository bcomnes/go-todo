package testdatabase

import "testing"

type fakeRunner struct {
	code int
	runs int
}

func (runner *fakeRunner) Run() int {
	runner.runs++
	return runner.code
}

func TestRunWithoutDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	runner := &fakeRunner{code: 7}
	if code := Run(runner); code != 7 || runner.runs != 1 {
		t.Fatalf("Run() = %d with %d runs", code, runner.runs)
	}
}

func TestQuoteIdentifier(t *testing.T) {
	if got := quoteIdentifier(`database"name`); got != `"database""name"` {
		t.Fatalf("quoteIdentifier() = %q", got)
	}
}
