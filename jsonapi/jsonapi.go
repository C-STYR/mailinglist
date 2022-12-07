package jsonapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mailinglist/mdb"
	"net/http"
)

// allows us to set headers and/or body for http responses
// here we're just setting the header Content-Type key
func setJsonHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

// convert JSON from the wire to a native go structure

func fromJson[T any](body io.Reader, target T) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(body)                   // buf <- body (syntax always seems backwards)
	json.Unmarshal(buf.Bytes(), &target) // converts bytes to the target data structure
}

/*
The anonymous withData function is a closure.
We'll be able to supply data (db connections, server status, etc) and keep them separate from the JSON
*/
func returnJson[T any](w http.ResponseWriter, withData func() (T, error)) {
	setJsonHeader(w) //set content-type etc.

	data, serverErr := withData() //returns the type, err

	if serverErr != nil {
		w.WriteHeader(500)                             // set the response to server error status
		serverErrJson, err := json.Marshal(&serverErr) // Marshall converts the error message to JSON
		if err != nil {
			log.Println(err)
			return
		}
		w.Write(serverErrJson) // write the json-type error and return
		return
	}

	dataJson, err := json.Marshal(&data) // convert the data to json
	if err != nil {
		log.Print(err)
		w.WriteHeader(500)
		return
	}
	w.Write(dataJson) // write the data to the body (will also call w.WriteHeader() with OK status first)
}

func returnErr(w http.ResponseWriter, err error, code int) {
	returnJson(w, func() (interface{}, error) { // interface here means any type is ok???
		errorMessage := struct { // creation of an anonymous struct
			Err string
		}{
			Err: err.Error(), // convert the passed err message to a string and assign to Err field
		}
		w.WriteHeader(code) // set the code to passed in code
		return errorMessage, nil
	})
}

func CreateEmail(db *sql.DB) http.Handler {
	// w: our writer, req: the incoming request we read from
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			return
		}
		entry := mdb.EmailEntry{}
		fromJson(req.Body, &entry) // read the req body and write it to the entry item

		// attempt to create the email in the database
		if err := mdb.CreateEmail(db, entry.Email); err != nil {
			returnErr(w, err, 400)
			return
		}

		// at this point the email is created
		returnJson(w, func() (interface{}, error) {
			log.Printf("JSON CreateEmail: %v\n", entry.Email) // log what we're doing
			return mdb.GetEmail(db, entry.Email)              // return the result of querying the database
		})
	})
}

func GetEmail(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			return
		}
		entry := mdb.EmailEntry{}
		fromJson(req.Body, &entry)

		returnJson(w, func() (interface{}, error) {
			log.Printf("JSON GetEmail: %v\n", entry.Email) // log what we're doing
			return mdb.GetEmail(db, entry.Email)           // return the function that actually access the databse
		})
	})
}

func UpdateEmail(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "PUT" {
			return
		}
		entry := mdb.EmailEntry{}
		fromJson(req.Body, &entry)

		// attempt to create the email in the database
		if err := mdb.UpdateEmail(db, entry); err != nil {
			returnErr(w, err, 400)
			return
		}

		// at this point the email is created
		returnJson(w, func() (interface{}, error) {
			log.Printf("JSON UpdateEmail: %v\n", entry.Email) // log what we're doing
			return mdb.GetEmail(db, entry.Email)              // return the result of querying the database
		})
	})
}

func DeleteEmail(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			return
		}
		entry := mdb.EmailEntry{}
		fromJson(req.Body, &entry)

		// attempt to create the email in the database
		if err := mdb.DeleteEmail(db, entry.Email); err != nil {
			returnErr(w, err, 400)
			return
		}

		// at this point the email is created
		returnJson(w, func() (interface{}, error) {
			log.Printf("JSON DeleteEmail: %v\n", entry.Email) // log what we're doing
			return mdb.GetEmail(db, entry.Email)              // return the result of querying the database
		})
	})
}

func GetEmailBatch(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			return
		}

		queryOptions := mdb.GetEmailBatchQueryParams{}
		fromJson(req.Body, &queryOptions)

		if queryOptions.Count <= 0 || queryOptions.Page <= 0 {
			returnErr(w, errors.New("page and count fields must be > 0"), 400)
		}

		returnJson(w, func() (interface{}, error) {
			log.Printf("JSON GetEmailBatch: %v\n", queryOptions)
			return mdb.GetEmailBatch(db, queryOptions)
		})
	})
}

func GetAllRows(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			return
		}

		returnJson(w, func() (interface{}, error) {
			log.Printf("JSON GetAllRows: running...")
			return "Results print in terminal!", mdb.GetAllRows(db)
		})
	})
}

func Serve(db *sql.DB, bind string) {
	// http.Handle creates a handler function at the specified endpoint using the passed in func
	// the passed in funcs map to the mdb.go analog
	http.Handle("/email/create", CreateEmail(db))
	http.Handle("/email/get", GetEmail(db))
	http.Handle("/email/get_batch", GetEmailBatch(db))
	http.Handle("/email/get_all", GetAllRows(db))
	http.Handle("/email/update", UpdateEmail(db))
	http.Handle("/email/delete", DeleteEmail(db))
	log.Printf("JSON API server listening on %v\n", bind)

	err := http.ListenAndServe(bind, nil) // our port
	if err != nil {
		log.Fatalf("JSON server error: %v", err) //if we can't listen, kill everything
	}
}
