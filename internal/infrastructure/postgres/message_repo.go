package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/dto"
	"github.com/jackc/pgx/v5"
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
        INSERT INTO messages (user_id, content, channel_id, direct_chat_id, client_msg_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, client_msg_id) DO NOTHING
		RETURNING id, user_id, content, channel_id, direct_chat_id, client_msg_id, created_at
    `, message.UserID, message.Content, message.ChannelID, message.DirectChatID, message.ClientMsgID).Scan(
		&message.ID,
		&message.UserID,
		&message.Content,
		&message.ChannelID,
		&message.DirectChatID,
		&message.ClientMsgID,
		&message.CreatedAt,
	)
	if err == nil {
		return message, nil // обычный путь: вставили новую строку
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Message{}, fmt.Errorf("create message: %w", err)
	}

	return r.findByClientMsgID(ctx, message.UserID, message.ClientMsgID)
}

func (r *MessageRepo) findByClientMsgID(ctx context.Context, userID string, clientMsgID *string) (domain.Message, error) {
	var message domain.Message
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, content, channel_id, direct_chat_id, client_msg_id, created_at
		FROM messages
		WHERE user_id = $1 AND client_msg_id = $2
	`, userID, clientMsgID).Scan(
		&message.ID,
		&message.UserID,
		&message.Content,
		&message.ChannelID,
		&message.DirectChatID,
		&message.ClientMsgID,
		&message.CreatedAt,
	)
	if err != nil {
		return domain.Message{}, fmt.Errorf("create message: find existing by client_msg_id: %w", err)
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

func (r *MessageRepo) GetByChannelID(ctx context.Context, channelID string, limit int, before time.Time) ([]domain.Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT * FROM (
    		SELECT id, user_id, content, channel_id, direct_chat_id, created_at
    		FROM messages
    		WHERE channel_id = $1
      			AND created_at < $2
    	ORDER BY created_at DESC
    	LIMIT $3
	) sub
	ORDER BY created_at ASC
	`, channelID, before, limit)
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

func (r *MessageRepo) GetByDirectChatID(ctx context.Context, directChatID string, limit int, before time.Time) ([]domain.Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT * FROM (
    		SELECT id, user_id, content, channel_id, direct_chat_id, created_at
    		FROM messages
    		WHERE direct_chat_id = $1
      			AND created_at < $2
    	ORDER BY created_at DESC
    	LIMIT $3
	) sub
	ORDER BY created_at ASC
	`, directChatID, before, limit)
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

func (r *MessageRepo) GetByChannelIDWithUsernames(
	ctx context.Context,
	channelID string,
	limit int,
	before time.Time,
) ([]dto.MessageResponse, error) {
	if before.IsZero() {

		before = time.Now().Add(time.Minute)
	}

	rows, err := r.db.Query(ctx, `
		SELECT * FROM (
			SELECT m.id, m.user_id, u.user_name AS username, m.content, m.created_at
			FROM messages m
			JOIN users u ON u.id = m.user_id
			WHERE m.channel_id = $1 AND m.created_at < $2
			ORDER BY m.created_at DESC
			LIMIT $3
		) sub
		ORDER BY created_at ASC
	`, channelID, before, limit)
	if err != nil {
		return nil, fmt.Errorf("query channel messages: %w", err)
	}
	defer rows.Close()

	var out []dto.MessageResponse
	for rows.Next() {
		var msg dto.MessageResponse
		if err := rows.Scan(
			&msg.ID,
			&msg.SenderID,
			&msg.Username,
			&msg.Content,
			&msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan channel message: %w", err)
		}
		out = append(out, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iter channel messages: %w", err)
	}
	return out, nil
}

func (r *MessageRepo) GetByDirectChatIDWithUsernames(
	ctx context.Context,
	directChatID string,
	limit int,
	before time.Time,
) ([]dto.MessageResponse, error) {
	if before.IsZero() {
		before = time.Now().Add(time.Minute)
	}

	rows, err := r.db.Query(ctx, `
		SELECT * FROM (
			SELECT m.id, m.user_id, u.user_name AS username, m.content, m.created_at
			FROM messages m
			JOIN users u ON u.id = m.user_id
			WHERE m.direct_chat_id = $1 AND m.created_at < $2
			ORDER BY m.created_at DESC
			LIMIT $3
		) sub
		ORDER BY created_at ASC
	`, directChatID, before, limit)
	if err != nil {
		return nil, fmt.Errorf("query dm messages: %w", err)
	}
	defer rows.Close()

	var out []dto.MessageResponse
	for rows.Next() {
		var msg dto.MessageResponse
		if err := rows.Scan(
			&msg.ID,
			&msg.SenderID,
			&msg.Username,
			&msg.Content,
			&msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan dm message: %w", err)
		}
		out = append(out, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iter dm messages: %w", err)
	}
	return out, nil
}
