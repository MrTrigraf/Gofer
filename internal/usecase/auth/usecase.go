package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	if len(username) == 0 || len(username) > 16 {
		return domain.User{}, domain.ErrUsernameIsLong
	}

	if len(password) < 6 {
		return domain.User{}, domain.ErrPasswordTooShort
	}
	if len(password) > 64 {
		return domain.User{}, domain.ErrPasswordTooLong
	}

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

func (uc *AuthUseCase) Login(ctx context.Context, username, password string) (domain.User, jwt.TokenPair, error) {
	user, err := uc.userRepo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// Юзера нет — отвечаем ровно тем же, чем на неверный пароль.
			return domain.User{}, jwt.TokenPair{}, domain.ErrInvalidCredentials
		}
		return domain.User{}, jwt.TokenPair{}, fmt.Errorf("login: find user: %w", err)
	}

	if err := uc.hasher.Compare(user.PasswordHash, password); err != nil {
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			slog.Error("login: password hash is unusable",
				"user_id", user.ID,
				"err", err,
			)
		}
		return domain.User{}, jwt.TokenPair{}, domain.ErrInvalidCredentials
	}

	tokens, err := uc.tokenService.GenerateTokens(user.ID, user.Username)
	if err != nil {
		return domain.User{}, jwt.TokenPair{}, fmt.Errorf("login: generate tokens: %w", err)
	}

	return user, tokens, nil
}

func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (jwt.TokenPair, error) {
	claims, err := uc.tokenService.ParseRefreshToken(refreshToken)
	if err != nil {
		return jwt.TokenPair{}, domain.ErrTokenInvalid
	}

	user, err := uc.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return jwt.TokenPair{}, domain.ErrUserNotFound
		}
		return jwt.TokenPair{}, fmt.Errorf("refresh token: find user: %w", err)
	}

	tokens, err := uc.tokenService.GenerateTokens(user.ID, user.Username)
	if err != nil {
		return jwt.TokenPair{}, fmt.Errorf("refresh token: generate tokens: %w", err)
	}

	return tokens, nil
}
