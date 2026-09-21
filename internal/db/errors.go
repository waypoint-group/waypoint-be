package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsForeignKeyViolation reports whether err violates the named foreign key constraint.
func IsForeignKeyViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == constraint
}

// IsUniqueViolation reports whether err violates the named unique constraint.
// Constraint names must match the explicit names in the database migrations.
func IsUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
