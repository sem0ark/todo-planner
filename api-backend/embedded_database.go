package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

const (
	embeddedDatabaseUsername = "todoplanner"
	embeddedDatabasePassword = "todoplanner"
	embeddedDatabaseName     = "todoplanner"
)

// EmbeddedDatabase owns the PostgreSQL process used by local desktop mode.
type EmbeddedDatabase struct {
	postgres      *embeddedpostgres.EmbeddedPostgres
	connectionURL string
}

// NewEmbeddedDatabase starts a PostgreSQL instance using a persistent local directory.
func NewEmbeddedDatabase(port uint32, dataDirectory string) (*EmbeddedDatabase, error) {
	resolvedDirectory, err := resolveDataDirectory(dataDirectory)
	if err != nil {
		return nil, fmt.Errorf("resolve embedded database directory: %w", err)
	}
	if err := os.MkdirAll(resolvedDirectory, 0o700); err != nil {
		return nil, fmt.Errorf("create embedded database directory: %w", err)
	}

	configuration := embeddedpostgres.DefaultConfig().
		Username(embeddedDatabaseUsername).
		Password(embeddedDatabasePassword).
		Database(embeddedDatabaseName).
		Port(port).
		RuntimePath(filepath.Join(resolvedDirectory, "runtime")).
		DataPath(filepath.Join(resolvedDirectory, "data")).
		BinariesPath(filepath.Join(resolvedDirectory, "binaries"))
	postgres := embeddedpostgres.NewDatabase(configuration)

	if err := postgres.Start(); err != nil {
		return nil, fmt.Errorf("start embedded PostgreSQL: %w", err)
	}

	return &EmbeddedDatabase{
		postgres:      postgres,
		connectionURL: configuration.GetConnectionURL(),
	}, nil
}

// ConnectionURL returns the connection string for the owned PostgreSQL process.
func (database *EmbeddedDatabase) ConnectionURL() string {
	return database.connectionURL
}

// Stop terminates the owned PostgreSQL process.
func (database *EmbeddedDatabase) Stop() error {
	if database == nil || database.postgres == nil {
		return nil
	}
	return database.postgres.Stop()
}

func resolveDataDirectory(dataDirectory string) (string, error) {
	if dataDirectory == "" {
		return "", fmt.Errorf("data directory is empty")
	}
	if dataDirectory == "~" || strings.HasPrefix(dataDirectory, "~/") {
		homeDirectory, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if dataDirectory == "~" {
			return homeDirectory, nil
		}
		return filepath.Join(homeDirectory, strings.TrimPrefix(dataDirectory, "~/")), nil
	}
	return filepath.Clean(dataDirectory), nil
}
