package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockChannelRepo struct {
	mock.Mock
}

type MockMessageRepo struct {
	mock.Mock
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

func TestHub_ChannelMessage(t *testing.T) {
	channelRepo := &MockChannelRepo{}
	messageRepo := &MockMessageRepo{}
	hub := NewHub(channelRepo, messageRepo)

	sender := &Client{
		userID: "user-1",
		send:   make(chan []byte, 1),
	}
	receiver := &Client{
		userID: "user-2",
		send:   make(chan []byte, 1),
	}

	hub.clients["user-1"] = sender
	hub.clients["user-2"] = receiver

	messageRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.Message")).
		Return(domain.Message{ID: "msg-1"}, nil)

	channelRepo.On("GetMembers", mock.Anything, "ch-1").
		Return([]domain.User{
			{ID: "user-1"},
			{ID: "user-2"},
		}, nil)

	msg := `{"type":"channel_message","target_id":"ch-1","content":"hello"}`

	go hub.Run()
	hub.broadcast <- IncomingMessage{
		client: sender,
		data:   []byte(msg),
	}

	select {
	case received := <-receiver.send:
		assert.Equal(t, []byte(msg), received)
	case <-time.After(time.Second):
		t.Fatal("receiver did not get message")
	}

	select {
	case <-sender.send:
		t.Fatal("sender should not receive own message")
	default:
	}
}

func TestHub_DMMessage(t *testing.T) {
	channelRepo := &MockChannelRepo{}
	messageRepo := &MockMessageRepo{}
	hub := NewHub(channelRepo, messageRepo)

	sender := &Client{
		userID: "user-1",
		send:   make(chan []byte, 1),
	}
	receiver := &Client{
		userID: "user-2",
		send:   make(chan []byte, 1),
	}

	hub.clients["user-1"] = sender
	hub.clients["user-2"] = receiver

	messageRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.Message")).
		Return(domain.Message{ID: "msg-1"}, nil)

	msg := `{"type":"dm_message","target_id":"dm-1","recipient_id":"user-2","content":"hello"}`

	go hub.Run()
	hub.broadcast <- IncomingMessage{
		client: sender,
		data:   []byte(msg),
	}

	select {
	case received := <-receiver.send:
		assert.Equal(t, []byte(msg), received)
	case <-time.After(time.Second):
		t.Fatal("receiver did not get message")
	}

	select {
	case <-sender.send:
		t.Fatal("sender should not receive own message")
	default:
	}
}
