package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofer/internal/domain"
	"github.com/gofer/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockUserRepo struct {
	mock.Mock
}

type MockHasher struct {
	mock.Mock
}

type MockTokenService struct {
	mock.Mock
}

func (m *MockUserRepo) Create(ctx context.Context, user domain.User) (domain.User, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(domain.User), args.Error(1)
}

func (m *MockUserRepo) FindByID(ctx context.Context, id string) (domain.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.User), args.Error(1)
}

func (m *MockUserRepo) FindByUsername(ctx context.Context, username string) (domain.User, error) {
	args := m.Called(ctx, username)
	return args.Get(0).(domain.User), args.Error(1)
}

func (m *MockUserRepo) SearchByUsername(ctx context.Context, query string) ([]domain.User, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *MockHasher) Hash(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockHasher) Compare(hash, password string) error {
	args := m.Called(hash, password)
	return args.Error(0)
}

func (m *MockTokenService) GenerateTokens(userID, username string) (jwt.TokenPair, error) {
	args := m.Called(userID, username)
	return args.Get(0).(jwt.TokenPair), args.Error(1)
}

func (m *MockTokenService) ParseAccessToken(token string) (*jwt.AccessClaims, error) {
	args := m.Called(token)
	return args.Get(0).(*jwt.AccessClaims), args.Error(1)
}

func (m *MockTokenService) ParseRefreshToken(token string) (*jwt.RefreshClaims, error) {
	args := m.Called(token)
	return args.Get(0).(*jwt.RefreshClaims), args.Error(1)
}

func TestRegister_Success(t *testing.T) {
	userRepo := &MockUserRepo{}
	hasher := &MockHasher{}
	tokenSvc := &MockTokenService{}
	uc := New(userRepo, hasher, tokenSvc)

	userRepo.On("FindByUsername", mock.Anything, "lol").
		Return(domain.User{}, domain.ErrNotFound)
	hasher.On("Hash", "secret123").
		Return("hashed_secret", nil)

	userRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.User")).
		Return(domain.User{ID: "123", Username: "lol"}, nil)

	user, err := uc.Register(context.Background(), "lol", "secret123")

	require.NoError(t, err)
	assert.Equal(t, "lol", user.Username)
	assert.Equal(t, "123", user.ID)
}

func TestRegister_AlreadyExists(t *testing.T) {
	userRepo := &MockUserRepo{}
	hasher := &MockHasher{}
	tokenSvc := &MockTokenService{}
	uc := New(userRepo, hasher, tokenSvc)

	userRepo.On("FindByUsername", mock.Anything, "lol").
		Return(domain.User{ID: "123", Username: "lol"}, nil)

	_, err := uc.Register(context.Background(), "lol", "secret123")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
}

func TestLogin_Success(t *testing.T) {
	userRepo := &MockUserRepo{}
	hasher := &MockHasher{}
	tokenSvc := &MockTokenService{}
	uc := New(userRepo, hasher, tokenSvc)

	userRepo.On("FindByUsername", mock.Anything, "lol").
		Return(domain.User{ID: "123", Username: "lol", PasswordHash: "hashed"}, nil)

	hasher.On("Compare", "hashed", "secret123").
		Return(nil)

	tokenSvc.On("GenerateTokens", "123", "lol").
		Return(jwt.TokenPair{AccessToken: "access", RefreshToken: "refresh"}, nil)

	_, tokens, err := uc.Login(context.Background(), "lol", "secret123")

	require.NoError(t, err)
	assert.Equal(t, "access", tokens.AccessToken)
	assert.Equal(t, "refresh", tokens.RefreshToken)
}

func TestLogin_WrongPassword(t *testing.T) {
	userRepo := &MockUserRepo{}
	hasher := &MockHasher{}
	tokenSvc := &MockTokenService{}
	uc := New(userRepo, hasher, tokenSvc)

	userRepo.On("FindByUsername", mock.Anything, "lol").
		Return(domain.User{ID: "123", Username: "lol", PasswordHash: "hashed"}, nil)

	hasher.On("Compare", "hashed", "error123").
		Return(errors.New("wrong password"))

	_, _, err := uc.Login(context.Background(), "lol", "error123")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestRegister_Validation(t *testing.T) {
	makeUC := func() *AuthUseCase {
		userRepo := &MockUserRepo{}
		hasher := &MockHasher{}
		tokenSvc := &MockTokenService{}

		// На случай, если вход валиден и дойдёт до репозитория/хешера.
		userRepo.On("FindByUsername", mock.Anything, mock.Anything).
			Return(domain.User{}, domain.ErrNotFound).Maybe()
		hasher.On("Hash", mock.Anything).Return("hashed", nil).Maybe()
		userRepo.On("Create", mock.Anything, mock.Anything).
			Return(domain.User{ID: "id-1"}, nil).Maybe()

		return New(userRepo, hasher, tokenSvc)
	}

	const validPass = "secret123"
	const validUser = "alice"

	tests := []struct {
		name     string
		username string
		password string
		wantErr  error // nil = ожидаем успех
	}{
		// --- username ---
		{"username empty", "", validPass, domain.ErrUsernameEmpty},
		{"username whitespace only", "   ", validPass, domain.ErrUsernameEmpty},
		{"username 16 latin ok", "abcdefghijklmnop", validPass, nil}, // 16 символов = 16 байт
		{"username 17 latin too long", "abcdefghijklmnopq", validPass, domain.ErrUsernameIsLong},
		{"username 16 cyrillic ok", "абвгдеёжзийклмно", validPass, nil}, // 16 символов, но 32 байта
		{"username valid", validUser, validPass, nil},

		// --- password (лимит = 72 БАЙТА, не символа — потолок bcrypt) ---
		{"password empty", validUser, "", domain.ErrPasswordTooShort},
		{"password too short", validUser, "12345", domain.ErrPasswordTooShort},
		{"password min ok", validUser, "123456", nil},                     // ровно 6 байт
		{"password 72 bytes ok", validUser, strings.Repeat("a", 72), nil}, // ровно 72 = предел
		{"password 73 bytes too long", validUser, strings.Repeat("a", 73), domain.ErrPasswordTooLong},
		// Многобайтовый край: 36 кириллических букв = 72 байта → ok (граница по БАЙТАМ).
		{"password 36 cyrillic ok (72 bytes)", validUser, strings.Repeat("а", 36), nil},
		// 37 кириллических = 74 байта → too long, хотя символов всего 37.
		{"password 37 cyrillic too long (74 bytes)", validUser, strings.Repeat("а", 37), domain.ErrPasswordTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := makeUC()
			_, err := uc.Register(context.Background(), tt.username, tt.password)

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
