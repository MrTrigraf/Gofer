package usecase

import (
	"context"

	"github.com/gofer/internal/domain"
	"github.com/gofer/pkg/jwt"
)

type UserRepository interface {
	Create(ctx context.Context, user domain.User) (domain.User, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
	FindByUsername(ctx context.Context, username string) (domain.User, error)
	SearchByUsername(ctx context.Context, query string) ([]domain.User, error)
}

type ChannelRepository interface {
	Create(ctx context.Context, channel domain.Channel) (domain.Channel, error)
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (domain.Channel, error)
	FindAll(ctx context.Context) ([]domain.Channel, error)
	FindByName(ctx context.Context, name string) (domain.Channel, error)
	AddMember(ctx context.Context, channelID string, userID string) error
	GetMembers(ctx context.Context, channelID string) ([]domain.User, error)
	RemoveMember(ctx context.Context, channelID string, userID string) error
	IsMember(ctx context.Context, channelID string, userID string) (bool, error)
}

type MessageRepository interface {
	Create(ctx context.Context, message domain.Message) (domain.Message, error)
	Delete(ctx context.Context, id string) error
	GetByChannelID(ctx context.Context, channelID string) ([]domain.Message, error)
	GetByDirectChatID(ctx context.Context, directChatID string) ([]domain.Message, error)
}

type DirectChatRepository interface {
	Create(ctx context.Context, direct domain.DirectChat) (domain.DirectChat, error)
	Delete(ctx context.Context, id string) error
	FindByUsers(ctx context.Context, user1ID string, user2ID string) (domain.DirectChat, error)
	FindByUserID(ctx context.Context, id string) ([]domain.DirectChat, error)
}

type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenService interface {
	GenerateTokens(userID, username string) (jwt.TokenPair, error)
	ParseAccessToken(tokenString string) (*jwt.AccessClaims, error)
	ParseRefreshToken(tokenString string) (*jwt.RefreshClaims, error)
}
