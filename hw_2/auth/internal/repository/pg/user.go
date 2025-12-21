package pg

import (
	"context"
	"database/sql"
	"fmt"

	"auth/internal/domain"
)


type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	const query = `
INSERT INTO users (id, email, name, password, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, email, name, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		user.ID, user.Email, user.Name, user.Password, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	const query = `
SELECT id, email, name, password, created_at, updated_at
FROM users
WHERE email = $1`

	var user domain.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	const query = `
SELECT id, email, name, password, created_at, updated_at
FROM users
WHERE id = $1`

	var user domain.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}