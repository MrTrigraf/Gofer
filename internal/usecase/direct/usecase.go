package direct

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofer/internal/domain"
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

func (uc *DirectUseCase) StartDM(user1ID, user2ID string) (domain.DirectChat, error) {
	_, err := uc.directRepo.FindByUsers(user1ID, user2ID)
	if err == nil {
		return domain.DirectChat{}, domain.ErrDirectChatAlreadyExists
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return domain.DirectChat{}, fmt.Errorf("start dm: check existing: %w", err)
	}

	_, err = uc.userRepo.FindByID(user1ID)
	if err != nil {
		return domain.DirectChat{}, domain.ErrUserNotFound
	}

	_, err = uc.userRepo.FindByID(user2ID)
	if err != nil {
		return domain.DirectChat{}, domain.ErrUserNotFound
	}

	direct := domain.DirectChat{
		UserID1:   user1ID,
		UserID2:   user2ID,
		CreatedAt: time.Now(),
	}

	direct, err = uc.directRepo.Create(direct)
	if err != nil {
		return domain.DirectChat{}, fmt.Errorf("start dm: %w", err)
	}

	return direct, nil
}

func (uc *DirectUseCase) DeleteDM(id string) error {
	err := uc.directRepo.Delete(id)
	if err != nil {
		return fmt.Errorf("delete direct: %w", err)
	}

	return nil
}

func (uc *DirectUseCase) GetDMHistory(directChatID string) ([]domain.Message, error) {
	messages, err := uc.messageRepo.GetByDirectChatID(directChatID)
	if err != nil {
		return nil, fmt.Errorf("list direct: %w", err)
	}

	return messages, nil
}
