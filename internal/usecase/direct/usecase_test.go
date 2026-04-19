package direct

import (
	"context"
	"testing"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockUserRepo struct {
	mock.Mock
}

type MockDirectRepo struct {
	mock.Mock
}

type MockMessageRepo struct {
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

func (m *MockDirectRepo) Create(ctx context.Context, direct domain.DirectChat) (domain.DirectChat, error) {
	args := m.Called(ctx, direct)
	return args.Get(0).(domain.DirectChat), args.Error(1)
}

func (m *MockDirectRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDirectRepo) FindByUsers(ctx context.Context, user1ID, user2ID string) (domain.DirectChat, error) {
	args := m.Called(ctx, user1ID, user2ID)
	return args.Get(0).(domain.DirectChat), args.Error(1)
}

func (m *MockDirectRepo) FindByUserID(ctx context.Context, id string) ([]domain.DirectChat, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]domain.DirectChat), args.Error(1)
}

func (m *MockMessageRepo) Create(ctx context.Context, message domain.Message) (domain.Message, error) {
	args := m.Called(ctx, message)
	return args.Get(0).(domain.Message), args.Error(1)
}

func (m *MockMessageRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMessageRepo) GetByChannelID(ctx context.Context, channelID string, limit int, before time.Time) ([]domain.Message, error) {
	args := m.Called(ctx, channelID, limit, before)
	return args.Get(0).([]domain.Message), args.Error(1)
}

func (m *MockMessageRepo) GetByDirectChatID(ctx context.Context, directChatID string, limit int, before time.Time) ([]domain.Message, error) {
	args := m.Called(ctx, directChatID, limit, before)
	return args.Get(0).([]domain.Message), args.Error(1)
}

func TestStartDM_Success(t *testing.T) {
	userRepo := &MockUserRepo{}
	directRepo := &MockDirectRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, directRepo, messageRepo)

	directRepo.On("FindByUsers", mock.Anything, "user-1", "user-2").
		Return(domain.DirectChat{}, domain.ErrNotFound)

	userRepo.On("FindByID", mock.Anything, "user-1").
		Return(domain.User{ID: "user-1"}, nil)

	userRepo.On("FindByID", mock.Anything, "user-2").
		Return(domain.User{ID: "user-2"}, nil)

	directRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.DirectChat")).
		Return(domain.DirectChat{ID: "dm-1", UserID1: "user-1", UserID2: "user-2"}, nil)

	dm, err := uc.StartDM(context.Background(), "user-1", "user-2")

	require.NoError(t, err)
	assert.Equal(t, "dm-1", dm.ID)
}

func TestStartDM_AlreadyExists(t *testing.T) {
	userRepo := &MockUserRepo{}
	directRepo := &MockDirectRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, directRepo, messageRepo)

	directRepo.On("FindByUsers", mock.Anything, "user-1", "user-2").
		Return(domain.DirectChat{ID: "dm-1"}, nil)

	_, err := uc.StartDM(context.Background(), "user-1", "user-2")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrDirectChatAlreadyExists)
}

func TestListDMs_Success(t *testing.T) {
	userRepo := &MockUserRepo{}
	directRepo := &MockDirectRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, directRepo, messageRepo)

	directRepo.On("FindByUserID", mock.Anything, "user-1").
		Return([]domain.DirectChat{
			{ID: "dm-1", UserID1: "user-1", UserID2: "user-2"},
			{ID: "dm-2", UserID1: "user-1", UserID2: "user-3"},
		}, nil)

	dms, err := uc.ListDMs(context.Background(), "user-1")

	require.NoError(t, err)
	assert.Len(t, dms, 2)
}
