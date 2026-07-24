package routes_test

import (
	"os"
	"testing"

	"github.com/bcomnes/go-todo/pkg/testdatabase"
)

func TestMain(tests *testing.M) {
	os.Exit(testdatabase.Run(tests))
}
