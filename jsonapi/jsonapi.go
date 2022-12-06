package jsonapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"io"
	"log"
)

// allows us to set headers and/or body for http responses
// here we're just setting the header Content-Type key
func setJsonHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

// convert JSON from the wire to a native go structure

func fromJson[T any](body io.Reader, target T) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(body) // buf <- body (syntax always seems backwards)
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
		w.WriteHeader(500) // set the response to server error status
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
	w. Write(dataJson) // write the data to the body (will also call w.WriteHeader() with OK status first)
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
