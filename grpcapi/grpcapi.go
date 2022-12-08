package grpcapi

import (
	"database/sql"
	"mailinglist/mdb"
	pb "mailinglist/proto"
	"time"
)

type MailServer struct {
	// embedding the below in any struct we create allows us to launch it with gRPC service
	pb.UnimplementedMailiningListServiceServer
	db *sql.DB
}

func pbEntryToMdbEntry(pbEntry *pb.EmailEntry) mdb.EmailEntry {
	// convert int64 to time.Time
	t := time.Unix(pbEntry.ConfirmedAt, 0)
	return mdb.EmailEntry{
		Id:          pbEntry.Id,
		Email:       pbEntry.Email,
		ConfirmedAt: &t,
		OptOut:      pbEntry.OptOut,
	}
}

func mdbEntryToPbEntry(mdbEntry *mdb.EmailEntry) pb.EmailEntry {
	return pb.EmailEntry{
		Id: mdbEntry.Id,
		Email: mdbEntry.Email,
		ConfirmedAt: mdbEntry.ConfirmedAt.Unix(),
		OptOut: mdbEntry.OptOut,
	}
}
