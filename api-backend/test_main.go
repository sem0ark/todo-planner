package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(testingMain *testing.M) {
	if os.Getenv("EMBEDDED_TEST_DATABASE") != "true" || os.Getenv("TEST_DATABASE_URL") != "" {
		os.Exit(testingMain.Run())
	}

	temporaryDirectory, err := os.MkdirTemp("", "todo-planner-test-postgres-")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create embedded test database directory: %v\n", err)
		os.Exit(1)
	}

	embeddedDatabase, err := NewEmbeddedDatabase(55433, filepath.Join(temporaryDirectory, "database"))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "start embedded test database: %v\n", err)
		_ = os.RemoveAll(temporaryDirectory)
		os.Exit(1)
	}
	if err := os.Setenv("TEST_DATABASE_URL", embeddedDatabase.ConnectionURL()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "configure embedded test database: %v\n", err)
		_ = embeddedDatabase.Stop()
		_ = os.RemoveAll(temporaryDirectory)
		os.Exit(1)
	}

	exitCode := testingMain.Run()
	if err := embeddedDatabase.Stop(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "stop embedded test database: %v\n", err)
		exitCode = 1
	}
	if err := os.RemoveAll(temporaryDirectory); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "remove embedded test database directory: %v\n", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}
