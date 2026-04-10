package auth

import (
	"fmt"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/usecase"
)

type AuthUseCase struct {
	userRepo usecase.UserRepository
	hasher   usecase.Hasher
}

func New(userRepo usecase.UserRepository, hasher usecase.Hasher) *AuthUseCase {
	return &AuthUseCase{
		userRepo: userRepo,
		hasher:   hasher,
	}
}

func (uc *AuthUseCase) Register(username, password string) (domain.User, error) {
	_, err := uc.userRepo.FindByUsername(username)
	if err == nil {
		return domain.User{}, domain.ErrUserAlreadyExists
	}

	hash, err := uc.hasher.Hash(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("register: hash password: %w", err)
	}

	user := domain.User{
		Username:     username,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}

	created, err := uc.userRepo.Create(user)
	if err != nil {
		return domain.User{}, fmt.Errorf("register: create user: %w", err)
	}

	return created, nil
}

func (uc *AuthUseCase) Login(username, password string) (domain.User, error) {
	user, err := uc.userRepo.FindByUsername(username)
	if err != nil {
		return domain.User{}, domain.ErrUserNotFound
	}

	if err = uc.hasher.Compare(user.PasswordHash, password); err != nil {
		return domain.User{}, domain.ErrInvalidCredentials
	}

	return user, nil
}

func (uc *AuthUseCase) RefreshToken(refreshToken string) (string, error) {
	//потом добавлю токен
	return "", nil
}
