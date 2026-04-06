package auth

import (
	"fmt"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/usecase"
	"golang.org/x/crypto/bcrypt"
)

type AuthUseCase struct {
	userRepo usecase.UserRepository
}

func New(userRepo usecase.UserRepository) *AuthUseCase {
	return &AuthUseCase{userRepo: userRepo}
}

func (uc *AuthUseCase) Register(username, password string) (domain.User, error) {
	_, err := uc.userRepo.FindByUsername(username)
	if err == nil {
		return domain.User{}, domain.ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, fmt.Errorf("register: hash password: %w", err)
	}

	user := domain.User{
		Username:     username,
		PasswordHash: string(hash),
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

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return domain.User{}, domain.ErrInvalidCredentials
	}

	return user, nil
}

func (uc *AuthUseCase) RefreshToken(refreshToken string) (string, error) {
	//потом добавлю токен
	return "", nil
}
