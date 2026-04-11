package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofer/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user domain.User) (domain.User, error) {
	err := r.db.QueryRow(ctx, `
        INSERT INTO users (user_name, password_hash, created_at)
        VALUES ($1, $2, $3)
        RETURNING id, user_name, password_hash, created_at
    `, user.Username, user.PasswordHash, user.CreatedAt).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (domain.User, error) {
	var user domain.User

	err := r.db.QueryRow(ctx, `
		SELECT id, user_name, password_hash, created_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("find user by id: %w", err)
	}

	return user, nil
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (domain.User, error) {
	var user domain.User

	err := r.db.QueryRow(ctx, `
		SELECT id, user_name, password_hash, created_at
		FROM users
		WHERE user_name = $1
	`, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("find user by username: %w", err)
	}

	return user, nil
}

func (r *UserRepo) SearchByUsername(ctx context.Context, query string) ([]domain.User, error) {
	rows, err := r.db.Query(ctx, `
	SELECT id, user_name, password_hash, created_at 
	FROM users 
	WHERE user_name ILIKE '%' || $1 || '%'
	`, query)
	if err != nil {
		return nil, fmt.Errorf("search users by username: %w", err)
	}
	defer rows.Close()

	var users []domain.User

	for rows.Next() {
		var user domain.User
		err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("search users by username: scan: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search users by username: rows: %w", err)
	}

	return users, nil
}
