package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"auth/proto"
	"gateway/internal/grpcjson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)


type Client struct {
	conn   *grpc.ClientConn
	client proto.AuthServiceClient
}

func NewClient(ctx context.Context) (*Client, func() error, error) {
	addr := getEnvOrDefault("AUTH_ADDR", "auth:50051")

	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(grpcjson.Codec{})),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial auth: %w", err)
	}

	client := proto.NewAuthServiceClient(conn)

	closeFn := func() error {
		return conn.Close()
	}

	return &Client{
		conn:   conn,
		client: client,
	}, closeFn, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Client) ValidateToken(ctx context.Context, token string) (string, error) {
	req := &proto.ValidateTokenRequest{
		Token: token,
	}

	resp, err := c.client.ValidateToken(ctx, req)
	if err != nil {
		return "", fmt.Errorf("validate token: %w", err)
	}

	if !resp.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return resp.UserId, nil
}

func (c *Client) Register(ctx context.Context, email, password, name string) (string, error) {
	resp, err := c.client.Register(ctx, &proto.RegisterRequest{
		Email:    email,
		Password: password,
		Name:     name,
	})
	if err != nil {
		return "", fmt.Errorf("register: %w", err)
	}
	return resp.UserId, nil
}

func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	resp, err := c.client.Login(ctx, &proto.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	return resp.Token, nil
}

