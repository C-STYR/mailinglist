package grpcapi

import (
	"context"
	"database/sql"
	"log"
	"mailinglist/mdb"
	pb "mailinglist/proto"
	"net"
	"time"

	"google.golang.org/grpc"
)

type MailServer struct {
	// embedding the below in any struct we create allows us to launch it with gRPC service
	pb.UnimplementedMailingListServiceServer
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
		Id:          mdbEntry.Id,
		Email:       mdbEntry.Email,
		ConfirmedAt: mdbEntry.ConfirmedAt.Unix(),
		OptOut:      mdbEntry.OptOut,
	}
}

// functions that must be implemented for UnimplementedMailingListServiceServer interface
func emailResponse(db *sql.DB, email string) (*pb.EmailResponse, error) {

	// make the database query
	entry, err := mdb.GetEmail(db, email)
	if err != nil {
		return &pb.EmailResponse{}, err
	}
	if entry == nil {
		return &pb.EmailResponse{}, nil
	}
	// convert to pb format (we have to use pb to send across the wire)
	res := mdbEntryToPbEntry(entry)
	return &pb.EmailResponse{EmailEntry: &res}, nil
}

// we have to include ctx for the interface spec, but it's unused here
func (s *MailServer) GetEmail(ctx context.Context, req *pb.GetEmailRequest) (*pb.EmailResponse, error) {
	log.Printf("gRPC GetEmail: %v\n", req)
	return emailResponse(s.db, req.EmailAddr) // queries and converts
}

func (s *MailServer) GetEmailBatch(ctx context.Context, req *pb.GetEmailBatchRequest) (*pb.GetEmailBatchResponse, error) {
	log.Printf("gRPC GetBatchEmail: %v\n", req)

	// convert params to the structure required by the db: GetEmailBatchQueryParams struct
	params := mdb.GetEmailBatchQueryParams{
		Page:  int(req.Page),
		Count: int(req.Count),
	}

	mdbEntries, err := mdb.GetEmailBatch(s.db, params)
	if err != nil {
		return &pb.GetEmailBatchResponse{}, err
	}

	// make only one memory allocation by setting the length to the number of mdbEntries
	pbEntries := make([]*pb.EmailEntry, 0, len(mdbEntries))
	for i := 0; i < len(mdbEntries); i++ {
		entry := mdbEntryToPbEntry(&mdbEntries[i])
		pbEntries = append(pbEntries, &entry)
	}

	return &pb.GetEmailBatchResponse{EmailEntries: pbEntries}, nil
}

func (s *MailServer) CreateEmail(ctx context.Context, req *pb.CreateEmailRequest) (*pb.EmailResponse, error) {
	log.Printf("gRPC CreateEmail: %v\n", req)

	err := mdb.CreateEmail(s.db, req.EmailAddr)
	if err != nil {
		return &pb.EmailResponse{}, err
	}

	return emailResponse(s.db, req.EmailAddr) // queries and converts
}

func (s *MailServer) UpdateEmail(ctx context.Context, req *pb.UpdateEmailRequest) (*pb.EmailResponse, error) {
	log.Printf("gRPC UpdateEmail: %v\n", req)

	entry := pbEntryToMdbEntry(req.EmailAddr)

	err := mdb.UpdateEmail(s.db, entry)
	if err != nil {
		return &pb.EmailResponse{}, err
	}

	return emailResponse(s.db, entry.Email) // queries and converts
}

func (s *MailServer) DeleteEmail(ctx context.Context, req *pb.DeleteEmailRequest) (*pb.EmailResponse, error) {
	log.Printf("gRPC UpdateEmail: %v\n", req)

	err := mdb.DeleteEmail(s.db, req.EmailAddr)
	if err != nil {
		return &pb.EmailResponse{}, err
	}

	// we're really updating the opt_out value, not deleting from database, so we return the entry
	return emailResponse(s.db, req.EmailAddr) // queries and converts
}

func Serve(db *sql.DB, bind string) {
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatalf("gRPC server error: failure to bind %v\n", bind)
	}
	grpcServer := grpc.NewServer()

	mailServer := MailServer{db: db}

	pb.RegisterMailingListServiceServer(grpcServer, &mailServer)

	log.Printf("gRPC API server listening on %v\n", bind)
	if err := grpcServer.Serve(listener); err != nil {
		//always kill if we get a server error
		log.Fatalf("gRPC server error: %v\n", err)
	}
}
