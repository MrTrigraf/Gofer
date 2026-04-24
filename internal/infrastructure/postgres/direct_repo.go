package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/dto"
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

func (r *DirectRepo) FindByID(ctx context.Context, id string) (domain.DirectChat, error) {
	var direct domain.DirectChat

	err := r.db.QueryRow(ctx, `
		SELECT id, user1_id, user2_id, created_at
		FROM direct_chats
		WHERE id = $1
	`, id).Scan(
		&direct.ID,
		&direct.UserID1,
		&direct.UserID2,
		&direct.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DirectChat{}, domain.ErrNotFound
		}
		return domain.DirectChat{}, fmt.Errorf("find direct by id: %w", err)
	}

	return direct, nil
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

func (r *DirectRepo) FindByUserIDWithUsernames(ctx context.Context, userID string) ([]dto.DirectChatResponse, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			dc.id,
			CASE WHEN dc.user1_id = $1 THEN dc.user2_id ELSE dc.user1_id END AS other_id,
			u.user_name,
			dc.created_at
		FROM direct_chats dc
		JOIN users u ON u.id = CASE
			WHEN dc.user1_id = $1 THEN dc.user2_id
			ELSE dc.user1_id
		END
		WHERE dc.user1_id = $1 OR dc.user2_id = $1
		ORDER BY dc.created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("find direct chats with usernames: %w", err)
	}
	defer rows.Close()

	var result []dto.DirectChatResponse

	for rows.Next() {
		var item dto.DirectChatResponse
		err := rows.Scan(
			&item.ID,
			&item.OtherUserID,
			&item.OtherUsername,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("find direct chats with usernames: scan: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find direct chats with usernames: rows: %w", err)
	}

	return result, nil
}
