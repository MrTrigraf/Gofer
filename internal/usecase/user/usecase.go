package user

import (
	"context"
	"fmt"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/usecase"
)

type UserUseCase struct {
	userRepo usecase.UserRepository
}

func New(userRepo usecase.UserRepository) *UserUseCase {
	return &UserUseCase{userRepo: userRepo}
}

func (uc *UserUseCase) GetProfile(ctx context.Context, id string) (domain.User, error) {
	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get profile: %w", err)
	}

	return user, nil
}

func (uc *UserUseCase) SearchUsers(ctx context.Context, username string) ([]domain.User, error) {
	users, err := uc.userRepo.SearchByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}

	return users, nil
}
