package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/usecase"
	"github.com/gofer/pkg/jwt"
)

type AuthUseCase struct {
	userRepo     usecase.UserRepository
	hasher       usecase.Hasher
	tokenService usecase.TokenService
}

func New(userRepo usecase.UserRepository, hasher usecase.Hasher, tokenService usecase.TokenService) *AuthUseCase {
	return &AuthUseCase{
		userRepo:     userRepo,
		hasher:       hasher,
		tokenService: tokenService,
	}
}

func (uc *AuthUseCase) Register(ctx context.Context, username, password string) (domain.User, error) {
	_, err := uc.userRepo.FindByUsername(ctx, username)
	if err == nil {
		return domain.User{}, domain.ErrUserAlreadyExists
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, fmt.Errorf("register: check username: %w", err)
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

	created, err := uc.userRepo.Create(ctx, user)
	if err != nil {
		return domain.User{}, fmt.Errorf("register: create user: %w", err)
	}

	return created, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, username, password string) (jwt.TokenPair, error) {
	user, err := uc.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return jwt.TokenPair{}, domain.ErrUserNotFound
	}

	if err = uc.hasher.Compare(user.PasswordHash, password); err != nil {
		return jwt.TokenPair{}, domain.ErrInvalidCredentials
	}

	return uc.tokenService.GenerateTokens(user.ID, user.Username)
}

func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (jwt.TokenPair, error) {
	claims, err := uc.tokenService.ParseRefreshToken(refreshToken)
	if err != nil {
		return jwt.TokenPair{}, domain.ErrTokenInvalid
	}

	user, err := uc.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return jwt.TokenPair{}, domain.ErrUserNotFound
	}

	return uc.tokenService.GenerateTokens(user.ID, user.Username)
}
