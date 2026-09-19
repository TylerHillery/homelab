package hlims

import (
	"context"
	"database/sql"
	"errors"

	"github.com/TylerHillery/homelab/services/hlims/db/migrations"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// SQLiteDB owns the HLIMS database and its generated queries.
type SQLiteDB struct {
	db      *sql.DB
	queries *database.Queries
}

// NewSQLiteDB opens an HLIMS database and applies all migrations.
func NewSQLiteDB(path string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, err
	}
	// Every SQLite connection gets a distinct :memory: database. Keep disposable
	// development and test databases on one connection so concurrent HTTP
	// requests observe the same schema and inventory.
	if path == ":memory:" {
		db.SetMaxOpenConns(1)
	}
	if err := db.Ping(); err != nil {
		return nil, errors.Join(err, db.Close())
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.Files)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	if _, err := provider.Up(context.Background()); err != nil {
		return nil, errors.Join(err, db.Close())
	}

	return &SQLiteDB{db: db, queries: database.New(db)}, nil
}

func sqliteDSN(path string) string {
	separator := "?"
	for _, char := range path {
		if char == '?' {
			separator = "&"
			break
		}
	}
	return path + separator + "_pragma=foreign_keys(1)"
}

// Close closes the database.
func (db *SQLiteDB) Close() error {
	return db.db.Close()
}
