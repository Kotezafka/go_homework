package auth

import (
	"context"
	"fmt"

	"auth/internal/domain"
	"auth/internal/service"
	"auth/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)


type Server struct {
	proto.UnimplementedAuthServiceServer
	service service.AuthService
}

func NewServer(svc service.AuthService) *Server {
	return &Server{service: svc}
}

func (s *Server) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	user, err := s.service.Register(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		if err == domain.ErrEmailAlreadyExist {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("register user: %v", err))
	}

	return &proto.RegisterResponse{
		UserId:  user.ID,
		Message: "user registered successfully",
	}, nil
}

func (s *Server) Login(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	token, err := s.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		if err == domain.ErrInvalidPassword {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("login: %v", err))
	}

	return &proto.LoginResponse{
		Token:     token,
		UserId:    "",
		ExpiresAt: "",
	}, nil
}

func (s *Server) ValidateToken(ctx context.Context, req *proto.ValidateTokenRequest) (*proto.ValidateTokenResponse, error) {
	userID, err := s.service.ValidateToken(ctx, req.Token)
	if err != nil {
		return &proto.ValidateTokenResponse{
			UserId: "",
			Valid:  false,
		}, nil
	}

	return &proto.ValidateTokenResponse{
		UserId: userID,
		Valid:  true,
	}, nil
}