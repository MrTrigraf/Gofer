package channel

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

type MockChannelRepo struct {
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

func (m *MockChannelRepo) Create(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	args := m.Called(ctx, channel)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *MockChannelRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChannelRepo) FindByID(ctx context.Context, id string) (domain.Channel, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *MockChannelRepo) FindAll(ctx context.Context) ([]domain.Channel, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Channel), args.Error(1)
}

func (m *MockChannelRepo) FindByName(ctx context.Context, name string) (domain.Channel, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *MockChannelRepo) AddMember(ctx context.Context, channelID string, userID string) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepo) GetMembers(ctx context.Context, channelID string) ([]domain.User, error) {
	args := m.Called(ctx, channelID)
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *MockChannelRepo) RemoveMember(ctx context.Context, channelID string, userID string) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepo) IsMember(ctx context.Context, channelID string, userID string) (bool, error) {
	args := m.Called(ctx, channelID, userID)
	return args.Bool(0), args.Error(1)
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

func TestCreateChannel_Success(t *testing.T) {
	userRepo := &MockUserRepo{}
	channelRepo := &MockChannelRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, channelRepo, messageRepo)

	channelRepo.On("FindByName", mock.Anything, "general").
		Return(domain.Channel{}, domain.ErrNotFound)

	channelRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.Channel")).
		Return(domain.Channel{ID: "ch-1", Name: "general"}, nil)

	ch, err := uc.CreateChannel(context.Background(), "general", "user-1")

	require.NoError(t, err)
	assert.Equal(t, "general", ch.Name)
	assert.Equal(t, "ch-1", ch.ID)
}

func TestCreateChannel_AlreadyExists(t *testing.T) {
	userRepo := &MockUserRepo{}
	channelRepo := &MockChannelRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, channelRepo, messageRepo)

	channelRepo.On("FindByName", mock.Anything, "general").
		Return(domain.Channel{ID: "ch-1", Name: "general"}, nil)

	_, err := uc.CreateChannel(context.Background(), "general", "user-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrChannelAlreadyExists)
}

func TestJoinChannel_Success(t *testing.T) {
	userRepo := &MockUserRepo{}
	channelRepo := &MockChannelRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, channelRepo, messageRepo)

	channelRepo.On("FindByID", mock.Anything, "ch-1").
		Return(domain.Channel{ID: "ch-1", Name: "general"}, nil)

	userRepo.On("FindByID", mock.Anything, "user-1").
		Return(domain.User{ID: "user-1", Username: "john"}, nil)

	channelRepo.On("AddMember", mock.Anything, "ch-1", "user-1").
		Return(nil)

	err := uc.JoinChannel(context.Background(), "ch-1", "user-1")

	require.NoError(t, err)
}

func TestListChannels_Success(t *testing.T) {
	userRepo := &MockUserRepo{}
	channelRepo := &MockChannelRepo{}
	messageRepo := &MockMessageRepo{}
	uc := New(userRepo, channelRepo, messageRepo)

	channelRepo.On("FindAll", mock.Anything).
		Return([]domain.Channel{
			{ID: "ch-1", Name: "general"},
			{ID: "ch-2", Name: "random"},
		}, nil)

	channels, err := uc.ListChannels(context.Background())

	require.NoError(t, err)
	assert.Len(t, channels, 2)
}
