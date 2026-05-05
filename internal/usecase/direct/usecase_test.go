package direct

import (
	"context"
	"testing"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/dto"
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

func (m *MockDirectRepo) FindByID(ctx context.Context, id string) (domain.DirectChat, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.DirectChat), args.Error(1)
}

func (m *MockDirectRepo) FindByUserIDWithUsernames(ctx context.Context, userID string) ([]dto.DirectChatResponse, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]dto.DirectChatResponse), args.Error(1)
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

	userRepo.On("FindByID", mock.Anything, "user-2").
		Return(domain.User{ID: "user-2"}, nil)

	directRepo.On("FindByUsers", mock.Anything, "user-1", "user-2").
		Return(domain.DirectChat{}, domain.ErrNotFound)

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

	userRepo.On("FindByID", mock.Anything, "user-2").
		Return(domain.User{ID: "user-2"}, nil)

	directRepo.On("FindByUsers", mock.Anything, "user-1", "user-2").
		Return(domain.DirectChat{ID: "dm-1"}, nil)

	_, err := uc.StartDM(context.Background(), "user-1", "user-2")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrDirectChatAlreadyExists)
}

func TestStartDM_TargetUserNotFound(t *testing.T) {
	userRepo := &MockUserRepo{}
	directRepo := &MockDirectRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, directRepo, messageRepo)

	userRepo.On("FindByID", mock.Anything, "user-missing").
		Return(domain.User{}, domain.ErrNotFound)

	_, err := uc.StartDM(context.Background(), "user-1", "user-missing")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
	directRepo.AssertNotCalled(t, "FindByUsers", mock.Anything, mock.Anything, mock.Anything)
	directRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestListDMs_Success(t *testing.T) {
	userRepo := &MockUserRepo{}
	directRepo := &MockDirectRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, directRepo, messageRepo)

	directRepo.On("FindByUserIDWithUsernames", mock.Anything, "user-1").
		Return([]dto.DirectChatResponse{
			{ID: "dm-1", OtherUserID: "user-2", OtherUsername: "alice"},
			{ID: "dm-2", OtherUserID: "user-3", OtherUsername: "bob"},
		}, nil)

	dms, err := uc.ListDMs(context.Background(), "user-1")

	require.NoError(t, err)
	assert.Len(t, dms, 2)
	assert.Equal(t, "alice", dms[0].OtherUsername)
	assert.Equal(t, "bob", dms[1].OtherUsername)
}

func TestDeleteDM_Success(t *testing.T) {
	userRepo := &MockUserRepo{}
	directRepo := &MockDirectRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, directRepo, messageRepo)

	directRepo.On("FindByID", mock.Anything, "dm-1").
		Return(domain.DirectChat{ID: "dm-1", UserID1: "user-1", UserID2: "user-2"}, nil)

	directRepo.On("Delete", mock.Anything, "dm-1").
		Return(nil)

	err := uc.DeleteDM(context.Background(), "dm-1", "user-1")

	require.NoError(t, err)
	directRepo.AssertCalled(t, "Delete", mock.Anything, "dm-1")
}

func TestDeleteDM_NotParticipant(t *testing.T) {
	userRepo := &MockUserRepo{}
	directRepo := &MockDirectRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, directRepo, messageRepo)

	directRepo.On("FindByID", mock.Anything, "dm-1").
		Return(domain.DirectChat{ID: "dm-1", UserID1: "user-1", UserID2: "user-2"}, nil)

	err := uc.DeleteDM(context.Background(), "dm-1", "user-3")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)
	directRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestDeleteDM_NotFound(t *testing.T) {
	userRepo := &MockUserRepo{}
	directRepo := &MockDirectRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, directRepo, messageRepo)

	directRepo.On("FindByID", mock.Anything, "dm-missing").
		Return(domain.DirectChat{}, domain.ErrNotFound)

	err := uc.DeleteDM(context.Background(), "dm-missing", "user-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrDirectChatNotFound)
	directRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}
