package data

import (
	"database/sql"
	"errors"
)

// ErrRecordNotFound is a custom error for when a movie can't be found in the database.
var (
	ErrRecordNotFound = errors.New("record not found")
)

// Models struct contains the other models our application needs.
type Models struct {
	Movies MovieModel
}

// NewModels returns a new Models struct.
func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
	}
}
