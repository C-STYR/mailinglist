package main

import (
	"database/sql"
	"log"
	"mailinglist/grpcapi"
	"mailinglist/jsonapi"
	"mailinglist/mdb"
	"sync"

	"github.com/alexflint/go-arg"
)

var args struct {
	DbPath   string `arg:"env:MAILINGLIST_DB"`
	BindJson string `arg:"env:MAILINGLIST_BIND_JSON"`
	BindGrpc string `arg:"env:MAILINGLIST_BIND_GRPC"`
}

func main() {
	arg.MustParse(&args)

	if args.DbPath == "" {
		args.DbPath = "list.db" // sets the default db as the included one in the filetree
	}

	if args.BindJson == "" {
		args.BindJson = ":8080"
	}

	if args.BindGrpc == "" {
		args.BindGrpc = ":8081"
	}

	log.Printf("using databse '%v'\n", args.DbPath)
	db, err := sql.Open("sqlite3", args.DbPath) // func Open(driverName, dataSourceName string) (*DB, error)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mdb.TryCreate(db) //create the table

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		log.Printf("starting JSON API server...\n")
		jsonapi.Serve(db, args.BindJson)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		log.Printf("starting GRPC API server...\n")
		grpcapi.Serve(db, args.BindGrpc)
		wg.Done()
	}()

	wg.Wait()
}
