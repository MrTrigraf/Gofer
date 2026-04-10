package channel

import (
	"context"
	"fmt"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/usecase"
)

type ChannelUseCase struct {
	userRepo    usecase.UserRepository
	channelRepo usecase.ChannelRepository
	messageRepo usecase.MessageRepository
}

func New(
	userRepo usecase.UserRepository,
	channelRepo usecase.ChannelRepository,
	messageRepo usecase.MessageRepository,
) *ChannelUseCase {
	return &ChannelUseCase{
		userRepo:    userRepo,
		channelRepo: channelRepo,
		messageRepo: messageRepo,
	}
}

func (uc *ChannelUseCase) CreateChannel(ctx context.Context, name string, createdBy string) (domain.Channel, error) {
	_, err := uc.channelRepo.FindByName(ctx, name)
	if err == nil {
		return domain.Channel{}, domain.ErrChannelAlreadyExists
	}

	channel := domain.Channel{
		Name:      name,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}

	created, err := uc.channelRepo.Create(ctx, channel)
	if err != nil {
		return domain.Channel{}, fmt.Errorf("create channel: %w", err)
	}

	return created, nil
}

func (uc *ChannelUseCase) JoinChannel(ctx context.Context, channelID string, userID string) error {
	_, err := uc.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return domain.ErrGroupNotFound
	}

	_, err = uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	err = uc.channelRepo.AddMember(ctx, channelID, userID)
	if err != nil {
		return fmt.Errorf("join channel: add member: %w", err)
	}

	return nil
}

func (uc *ChannelUseCase) LeaveChannel(ctx context.Context, channelID string, userID string) error {
	_, err := uc.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return domain.ErrGroupNotFound
	}

	_, err = uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	err = uc.channelRepo.RemoveMember(ctx, channelID, userID)
	if err != nil {
		return fmt.Errorf("leave channel: remove member: %w", err)
	}

	return nil
}

func (uc *ChannelUseCase) ListChannels(ctx context.Context) ([]domain.Channel, error) {
	channels, err := uc.channelRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}

	return channels, nil
}

func (uc *ChannelUseCase) DeleteChannel(ctx context.Context, id string) error {
	err := uc.channelRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}

	return nil
}

func (uc *ChannelUseCase) GetChannelHistory(ctx context.Context, channelID string) ([]domain.Message, error) {
	messages, err := uc.messageRepo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("get channel history: %w", err)
	}

	return messages, nil
}
