package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofer/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DirectRepo struct {
	db *pgxpool.Pool
}

func NewDirectRepo(db *pgxpool.Pool) *DirectRepo {
	return &DirectRepo{db: db}
}

func (r *DirectRepo) Create(ctx context.Context, direct domain.DirectChat) (domain.DirectChat, error) {
	err := r.db.QueryRow(ctx, `
        INSERT INTO direct_chats (user1_id, user2_id, created_at)
		VALUES ($1, $2, $3)
		RETURNING id, user1_id, user2_id, created_at
    `, direct.UserID1, direct.UserID2, direct.CreatedAt).Scan(
		&direct.ID,
		&direct.UserID1,
		&direct.UserID2,
		&direct.CreatedAt,
	)
	if err != nil {
		return domain.DirectChat{}, fmt.Errorf("create direct: %w", err)
	}

	return direct, nil
}

func (r *DirectRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM direct_chats WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete direct: %w", err)
	}

	return nil
}

func (r *DirectRepo) FindByUsers(ctx context.Context, user1ID, user2ID string) (domain.DirectChat, error) {
	var direct domain.DirectChat

	err := r.db.QueryRow(ctx, `
		SELECT id, user1_id, user2_id, created_at
		FROM direct_chats
		WHERE (user1_id = $1 AND user2_id = $2)
		   OR (user1_id = $2 AND user2_id = $1)
	`, user1ID, user2ID).Scan(
		&direct.ID,
		&direct.UserID1,
		&direct.UserID2,
		&direct.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DirectChat{}, domain.ErrNotFound
		}
		return domain.DirectChat{}, fmt.Errorf("find direct by users: %w", err)
	}

	return direct, nil
}

func (r *DirectRepo) FindByUserID(ctx context.Context, id string) ([]domain.DirectChat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user1_id, user2_id, created_at
		FROM direct_chats
		WHERE user1_id = $1 OR user2_id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("find directs by user: %w", err)
	}
	defer rows.Close()

	var directs []domain.DirectChat

	for rows.Next() {
		var direct domain.DirectChat
		err := rows.Scan(
			&direct.ID,
			&direct.UserID1,
			&direct.UserID2,
			&direct.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("find directs by user: scan: %w", err)
		}
		directs = append(directs, direct)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find directs by user: rows: %w", err)
	}

	return directs, nil
}
