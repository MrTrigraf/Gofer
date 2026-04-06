package user

import (
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

func (uc *UserUseCase) GetProfile(id string) (domain.User, error) {
	user, err := uc.userRepo.FindByID(id)
	if err != nil {
		return domain.User{}, domain.ErrUserNotFound
	}

	return user, nil
}

func (uc *UserUseCase) SearchUsers(username string) ([]domain.User, error) {
	users, err := uc.userRepo.SearchByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}

	return users, nil
}
