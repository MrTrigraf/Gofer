package postgres

import (
	"context"
	"fmt"

	"github.com/gofer/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepo struct {
	db *pgxpool.Pool
}

func NewMessageRepo(db *pgxpool.Pool) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) Create(ctx context.Context, message domain.Message) (domain.Message, error) {
	err := r.db.QueryRow(ctx, `
        INSERT INTO messages (user_id, content, channel_id, direct_chat_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, content, channel_id, direct_chat_id, created_at
    `, message.UserID, message.Content, &message.ChannelID, &message.DirectChatID, message.CreatedAt).Scan(
		&message.ID,
		&message.UserID,
		&message.Content,
		&message.ChannelID,
		&message.DirectChatID,
		&message.CreatedAt,
	)
	if err != nil {
		return domain.Message{}, fmt.Errorf("create message: %w", err)
	}

	return message, nil
}

func (r *MessageRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM messages WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	return nil
}

func (r *MessageRepo) GetByChannelID(ctx context.Context, channelID string) ([]domain.Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, content, channel_id, direct_chat_id, created_at
		FROM messages
		WHERE channel_id = $1
		ORDER BY created_at ASC
	`, channelID)
	if err != nil {
		return nil, fmt.Errorf("get messages by channel: %w", err)
	}
	defer rows.Close()

	var messages []domain.Message

	for rows.Next() {
		var message domain.Message
		err := rows.Scan(
			&message.ID,
			&message.UserID,
			&message.Content,
			&message.ChannelID,
			&message.DirectChatID,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("get messages by channel: scan: %w", err)
		}
		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get messages by channel: rows: %w", err)
	}

	return messages, nil
}

func (r *MessageRepo) GetByDirectChatID(ctx context.Context, directChatID string) ([]domain.Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, content, channel_id, direct_chat_id, created_at
		FROM messages
		WHERE direct_chat_id = $1
		ORDER BY created_at ASC
	`, directChatID)
	if err != nil {
		return nil, fmt.Errorf("get messages by direct chat: %w", err)
	}
	defer rows.Close()

	var messages []domain.Message

	for rows.Next() {
		var message domain.Message
		err := rows.Scan(
			&message.ID,
			&message.UserID,
			&message.Content,
			&message.ChannelID,
			&message.DirectChatID,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("get messages by direct chat: scan: %w", err)
		}
		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get messages by direct chat: rows: %w", err)
	}

	return messages, nil
}
