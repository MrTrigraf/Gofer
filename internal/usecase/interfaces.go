package usecase

import "github.com/gofer/internal/domain"

type UserRepository interface {
	Create(user domain.User) (domain.User, error)
	FindByID(id string) (domain.User, error)
	FindByUsername(username string) (domain.User, error)
	SearchByUsername(query string) ([]domain.User, error)
}

type ChannelRepository interface {
	Create(channel domain.Channel) (domain.Channel, error)
	Delete(id string) error
	FindByID(id string) (domain.Channel, error)
	FindAll() ([]domain.Channel, error)
	FindByName(name string) (domain.Channel, error)
	AddMember(channelID string, userID string) error
	GetMembers(channelID string) ([]domain.User, error)
	RemoveMember(channelID string, userID string) error
	IsMember(channelID string, userID string) (bool, error)
}

type MessageRepository interface {
	Create(message domain.Message) (domain.Message, error)
	Delete(id string) error
	GetByChannelID(channelID string) ([]domain.Message, error)
	GetByDirectChatID(directChatID string) ([]domain.Message, error)
}

type DirectChatRepository interface {
	Create(direct domain.DirectChat) (domain.DirectChat, error)
	Delete(id string) error
	FindByUsers(user1ID string, user2ID string) (domain.DirectChat, error)
	FindByUserID(id string) ([]domain.DirectChat, error)
}

type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}
