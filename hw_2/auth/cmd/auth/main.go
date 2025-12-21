package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	"auth/internal/repository/pg"
	"auth/internal/service"
	"auth/internal/grpcjson"
	"auth/pkg/auth"
	"auth/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:password@localhost:5432/auth?sslmode=disable"
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	userRepo := pg.NewUserRepository(db)

	authSvc := service.New(userRepo)

	grpcServer := grpc.NewServer(grpc.ForceServerCodec(grpcjson.Codec{}))

	authServer := auth.NewServer(authSvc)
	proto.RegisterAuthServiceServer(grpcServer, authServer)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	fmt.Println("Auth service started on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}