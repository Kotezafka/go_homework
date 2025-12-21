package main

import (
	"context"
	"log"
	"net"

	"ledger/internal/app"
	"ledger/internal/grpcjson"
	"ledger/pkg/ledger"
	"ledger/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx := context.Background()

	svc, closeFn, err := app.NewService(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}
	defer func() {
		if err := closeFn(); err != nil {
			log.Printf("Error closing service: %v", err)
		}
	}()

	grpcServer := grpc.NewServer(grpc.ForceServerCodec(grpcjson.Codec{}))

	ledgerServer := ledger.NewServer(svc)
	proto.RegisterLedgerServiceServer(grpcServer, ledgerServer)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Println("Ledger service started on :50052")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}