package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofer/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChannelRepo struct {
	db *pgxpool.Pool
}

func NewChannelRepo(db *pgxpool.Pool) *ChannelRepo {
	return &ChannelRepo{db: db}
}

func (r *ChannelRepo) Create(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	err := r.db.QueryRow(ctx, `
        INSERT INTO channels (channel_name, created_by, created_at)
        VALUES ($1, $2, $3)
        RETURNING id, channel_name, created_by, created_at
    `, channel.Name, channel.CreatedBy, channel.CreatedAt).Scan(
		&channel.ID,
		&channel.Name,
		&channel.CreatedBy,
		&channel.CreatedAt,
	)
	if err != nil {
		return domain.Channel{}, fmt.Errorf("create channel: %w", err)
	}

	return channel, nil
}

func (r *ChannelRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM channels WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}

	return nil
}

func (r *ChannelRepo) FindByID(ctx context.Context, id string) (domain.Channel, error) {
	var channel domain.Channel

	err := r.db.QueryRow(ctx, `
		SELECT id, channel_name, created_by, created_at
		FROM channels
		WHERE id = $1
	`, id).Scan(
		&channel.ID,
		&channel.Name,
		&channel.CreatedBy,
		&channel.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Channel{}, domain.ErrNotFound
		}
		return domain.Channel{}, fmt.Errorf("find channel by id: %w", err)
	}

	return channel, nil
}

func (r *ChannelRepo) FindAll(ctx context.Context) ([]domain.Channel, error) {
	rows, err := r.db.Query(ctx, `
	SELECT id, channel_name, created_by, created_at 
	FROM channels
	`)
	if err != nil {
		return nil, fmt.Errorf("find all channels: %w", err)
	}
	defer rows.Close()

	var channels []domain.Channel

	for rows.Next() {
		var channel domain.Channel
		err := rows.Scan(&channel.ID, &channel.Name, &channel.CreatedBy, &channel.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("find all channels: scan: %w", err)
		}
		channels = append(channels, channel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find all channels: %w", err)
	}

	return channels, nil
}

func (r *ChannelRepo) FindByName(ctx context.Context, name string) (domain.Channel, error) {
	var channel domain.Channel

	err := r.db.QueryRow(ctx, `
		SELECT id, channel_name, created_by, created_at
		FROM channels
		WHERE channel_name = $1
	`, name).Scan(
		&channel.ID,
		&channel.Name,
		&channel.CreatedBy,
		&channel.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Channel{}, domain.ErrNotFound
		}
		return domain.Channel{}, fmt.Errorf("find channel by name: %w", err)
	}

	return channel, nil
}

func (r *ChannelRepo) AddMember(ctx context.Context, channelID, userID string) error {
	_, err := r.db.Exec(ctx, `
        INSERT INTO channel_members (channel_id, user_id)
        VALUES ($1, $2)
    `, channelID, userID)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	return nil
}

func (r *ChannelRepo) GetMembers(ctx context.Context, channelID string) ([]domain.User, error) {
	rows, err := r.db.Query(ctx, `
	SELECT u.id, u.user_name, u.password_hash, u.created_at
	FROM users u
	JOIN channel_members cm ON cm.user_id = u.id
	WHERE cm.channel_id = $1
	`, channelID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}
	defer rows.Close()

	var users []domain.User

	for rows.Next() {
		var user domain.User
		err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("get members: scan: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}

	return users, nil
}

func (r *ChannelRepo) RemoveMember(ctx context.Context, channelID, userID string) error {
	_, err := r.db.Exec(ctx, `
        DELETE FROM channel_members 
		WHERE channel_id = $1 AND user_id = $2
    `, channelID, userID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	return nil
}

func (r *ChannelRepo) IsMember(ctx context.Context, channelID, userID string) (bool, error) {
	var exists bool

	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
    	SELECT 1 FROM channel_members 
    	WHERE channel_id = $1 AND user_id = $2
		)
    `, channelID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is member: %w", err)
	}

	return exists, nil
}
