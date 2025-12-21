package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"auth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	tokenExpiration = 24 * time.Hour
	jwtSecret       = "your-secret-key-change-in-production"
)

type AuthService interface {
	Register(ctx context.Context, email, password, name string) (domain.User, error)
	Login(ctx context.Context, email, password string) (string, error)
	ValidateToken(ctx context.Context, token string) (string, error)
}

type authService struct {
	userRepo domain.UserRepository
}

func New(userRepo domain.UserRepository) AuthService {
	return &authService{
		userRepo: userRepo,
	}
}

func (s *authService) Register(ctx context.Context, email, password, name string) (domain.User, error) {
	_, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil {
		return domain.User{}, domain.ErrEmailAlreadyExist
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, fmt.Errorf("check existing user: %w", err)
	}

	user := domain.User{
		ID:        uuid.New().String(),
		Email:     email,
		Name:      name,
		Password:  password,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := user.Validate(); err != nil {
		return domain.User{}, fmt.Errorf("validate user: %w", err)
	}

	if err := user.HashPassword(); err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}

	created, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return created, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrInvalidPassword
		}
		return "", fmt.Errorf("get user: %w", err)
	}

	if !user.CheckPassword(password) {
		return "", domain.ErrInvalidPassword
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return token, nil
}

func (s *authService) ValidateToken(ctx context.Context, token string) (string, error) {
	claims, err := s.parseToken(token)
	if err != nil {
		return "", fmt.Errorf("parse token: %w", err)
	}

	return claims.UserID, nil
}

func (s *authService) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(tokenExpiration).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func (s *authService) parseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}