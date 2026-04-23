package direct

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/dto"
	"github.com/gofer/internal/usecase"
)

type DirectUseCase struct {
	userRepo    usecase.UserRepository
	directRepo  usecase.DirectChatRepository
	messageRepo usecase.MessageRepository
}

func New(userRepo usecase.UserRepository, directRepo usecase.DirectChatRepository, messageRepo usecase.MessageRepository) *DirectUseCase {
	return &DirectUseCase{
		userRepo:    userRepo,
		directRepo:  directRepo,
		messageRepo: messageRepo,
	}
}

func (uc *DirectUseCase) StartDM(ctx context.Context, user1ID, user2ID string) (domain.DirectChat, error) {
	_, err := uc.directRepo.FindByUsers(ctx, user1ID, user2ID)
	if err == nil {
		return domain.DirectChat{}, domain.ErrDirectChatAlreadyExists
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return domain.DirectChat{}, fmt.Errorf("start dm: check existing: %w", err)
	}

	_, err = uc.userRepo.FindByID(ctx, user1ID)
	if err != nil {
		return domain.DirectChat{}, domain.ErrUserNotFound
	}

	_, err = uc.userRepo.FindByID(ctx, user2ID)
	if err != nil {
		return domain.DirectChat{}, domain.ErrUserNotFound
	}

	direct := domain.DirectChat{
		UserID1:   user1ID,
		UserID2:   user2ID,
		CreatedAt: time.Now(),
	}

	direct, err = uc.directRepo.Create(ctx, direct)
	if err != nil {
		return domain.DirectChat{}, fmt.Errorf("start dm: %w", err)
	}

	return direct, nil
}

func (uc *DirectUseCase) DeleteDM(ctx context.Context, id string) error {
	err := uc.directRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete direct: %w", err)
	}

	return nil
}

func (uc *DirectUseCase) GetDMHistory(ctx context.Context, directChatID string, limit int, before time.Time) ([]domain.Message, error) {
	messages, err := uc.messageRepo.GetByDirectChatID(ctx, directChatID, limit, before)
	if err != nil {
		return nil, fmt.Errorf("list direct: %w", err)
	}

	return messages, nil
}

func (uc *DirectUseCase) ListDMs(ctx context.Context, userID string) ([]dto.DirectChatResponse, error) {
	dms, err := uc.directRepo.FindByUserIDWithUsernames(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list dms: %w", err)
	}
	return dms, nil
}
