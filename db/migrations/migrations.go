// Package migrations provides access to database migration files.
package migrations

import "embed"

// FS contains the embedded SQL migrations.
//
//go:embed *.sql
var FS embed.FS
