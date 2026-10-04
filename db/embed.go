// Package db exposes the goose migrations to Go code (integration tests), so they don't depend on the working directory.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
