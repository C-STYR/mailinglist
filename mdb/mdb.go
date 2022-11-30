package mdb

import (
	"database/sql"
	"log"
	"time"
	"github.com/mattn/go-sqlite3"
)

type EmailEntry struct {
	Id int64
	EmailEntry string
	ConfirmedAt time.Time
	OptOut bool
}

// func TryCreate(db *sql.DB) {
// 	_, err := db.Exec(`

// 	`)
// }