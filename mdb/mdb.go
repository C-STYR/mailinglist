package mdb

import (
	"database/sql"
	"log"
	"time"

	"github.com/mattn/go-sqlite3"
)

type EmailEntry struct {
	Id          int64
	Email       string
	ConfirmedAt *time.Time
	OptOut      bool
}

func TryCreate(db *sql.DB) { //this functionality not tied to a specific db, instead takes a pointer
	_, err := db.Exec(`
		CREATE TABLE emails (
			id 	INTEGER PRIMARY KEY,
			email	TEXT UNIQUE,
			confirmed_at INTEGER,
			opt_out INTEGER
		);
	`)
	if err != nil {
		if sqlError, ok := err.(sqlite3.Error); ok {
			// code 1 == "table already exists"
			if sqlError.Code != 1 {
				log.Fatal(sqlError)
			}
		} else {
			log.Fatal(err)
		}
	}
}

func emailEntryFromRow(row *sql.Rows) (*EmailEntry, error) {
	var id int64
	var email string
	var confirmedAt int64
	var optOut bool

	err := row.Scan(&id, &email, &confirmedAt, &optOut) // the number of values passed in must be the number of columns in the table

	if err != nil {
		log.Println(err)
		return nil, err
	}

	t := time.Unix(confirmedAt, 0)

	return &EmailEntry{Id: id, Email: email, ConfirmedAt: &t, OptOut: optOut}, nil
}

func CreateEmail(db *sql.DB, email string) error {

	// we're only concerned with the error, we don't need the return value "Result"
	_, err := db.Exec(` 
	INSERT INTO
	emails(email, confirmed_at, opt_out)
	VALUES(?, 0, false)`, email)
	// db.Exec(query string, args ...any) (Result, error)
	// email is substituted for "?"
	// sqlite3 will set the id automatically (no need for a generate function)

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func GetEmail(db *sql.DB, email string) (*EmailEntry, error) {

	// db.Query works much like .Exec but DOES return rows
	rows, err := db.Query(`
	SELECT id, email, confirmed_at, opt_out
	FROM emails
	WHERE email = ?`, email)

	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close() //unlike .Exec, .Query leaves the return value open,

	for rows.Next() {
		// .Next() scans through the rows returned from the query which have sql.Rows data type (see emailEntryFromRow signature)
		return emailEntryFromRow(rows) //emails are unique, so should return only one EmailEntry
	}

	// if there are no hits returned from the query, this handles both return values
	return nil, nil
}

func UpdateEmail(db *sql.DB, entry EmailEntry) error {
	t := entry.ConfirmedAt.Unix()

	_, err := db.Exec(`
	INSERT INTO
	emails(email, confirmed_at, opt_out)
	VALUES(?, ?, ?)
	ON CONFLICT(email) DO UPDATE SET
		confirmed_at=?
		opt_out=?`, entry.Email, t, entry.OptOut, t, entry.OptOut) //5x args for 5x ?s

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

/*
Why don't we actually delete?
Setting opt_out to false prevents the email from being re-added either maliciously or accidentally
We retain the entry, allowing the user to re-subscribe without re-registering
*/
func DeleteEmail(db *sql.DB, email string) error {
	_, err := db.Exec(`
	UPDATE emails
	SET opt_out=true
	WHERE email=?`, email)

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

type GetEmailBatchQueryParams struct {
	Page  int // for pagination
	Count int
}

func GetEmailBatch(db *sql.DB, params GetEmailBatchQueryParams) ([]EmailEntry, error) {
	var empty []EmailEntry // an empty slice

	rows, err := db.Query(`
	SELECT id, email, confirmed_at, opt_out
	FROM emails
	WHERE opt_out = false
	ORDER BY id ASC
	LIMIT ? OFFSET ?`, params.Count, (params.Page-1)*params.Count) //offset works on an index, but pages start at 1

	if err != nil {
		log.Println(err)
		return empty, err
	}
	defer rows.Close()

	/*
		An optimization:
		By setting the capacity to the expected number of results,
		we only need one memory allocation for the slice
	*/
	emails := make([]EmailEntry, 0, params.Count)

	for rows.Next() {
		email, err := emailEntryFromRow(rows)
		if err != nil {
			// no partial list returned: if err, return nil
			return nil, err
		}
		emails = append(emails, *email)
	}
	return emails, nil
}
