package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofer/internal/domain"
	"github.com/gofer/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockChannelRepo struct{ mock.Mock }

func (m *MockChannelRepo) CreateWithMember(ctx context.Context, channel domain.Channel, userID string) (domain.Channel, error) {
	args := m.Called(ctx, channel, userID)
	return args.Get(0).(domain.Channel), args.Error(1)
}
func (m *MockChannelRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockChannelRepo) FindByID(ctx context.Context, id string) (domain.Channel, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Channel), args.Error(1)
}
func (m *MockChannelRepo) FindByUserID(ctx context.Context, userID string) ([]domain.Channel, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.Channel), args.Error(1)
}

func (m *MockChannelRepo) AddMember(ctx context.Context, channelID, userID string) error {
	return m.Called(ctx, channelID, userID).Error(0)
}
func (m *MockChannelRepo) GetMembers(ctx context.Context, channelID string) ([]domain.User, error) {
	args := m.Called(ctx, channelID)
	return args.Get(0).([]domain.User), args.Error(1)
}
func (m *MockChannelRepo) RemoveMember(ctx context.Context, channelID, userID string) error {
	return m.Called(ctx, channelID, userID).Error(0)
}
func (m *MockChannelRepo) IsMember(ctx context.Context, channelID, userID string) (bool, error) {
	args := m.Called(ctx, channelID, userID)
	return args.Bool(0), args.Error(1)
}

type MockMessageRepo struct{ mock.Mock }

func (m *MockMessageRepo) Create(ctx context.Context, message domain.Message) (domain.Message, error) {
	args := m.Called(ctx, message)
	return args.Get(0).(domain.Message), args.Error(1)
}
func (m *MockMessageRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockMessageRepo) GetByChannelID(ctx context.Context, channelID string, limit int, before time.Time) ([]domain.Message, error) {
	args := m.Called(ctx, channelID, limit, before)
	return args.Get(0).([]domain.Message), args.Error(1)
}
func (m *MockMessageRepo) GetByDirectChatID(ctx context.Context, directChatID string, limit int, before time.Time) ([]domain.Message, error) {
	args := m.Called(ctx, directChatID, limit, before)
	return args.Get(0).([]domain.Message), args.Error(1)
}

func (m *MockMessageRepo) GetByChannelIDWithUsernames(ctx context.Context, channelID string, limit int, before time.Time) ([]dto.MessageResponse, error) {
	args := m.Called(ctx, channelID, limit, before)
	return args.Get(0).([]dto.MessageResponse), args.Error(1)
}

func (m *MockMessageRepo) GetByDirectChatIDWithUsernames(ctx context.Context, directChatID string, limit int, before time.Time) ([]dto.MessageResponse, error) {
	args := m.Called(ctx, directChatID, limit, before)
	return args.Get(0).([]dto.MessageResponse), args.Error(1)
}

type MockDirectRepo struct{ mock.Mock }

func (m *MockDirectRepo) Create(ctx context.Context, d domain.DirectChat) (domain.DirectChat, error) {
	args := m.Called(ctx, d)
	return args.Get(0).(domain.DirectChat), args.Error(1)
}
func (m *MockDirectRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockDirectRepo) FindByID(ctx context.Context, id string) (domain.DirectChat, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.DirectChat), args.Error(1)
}
func (m *MockDirectRepo) FindByUsers(ctx context.Context, u1, u2 string) (domain.DirectChat, error) {
	args := m.Called(ctx, u1, u2)
	return args.Get(0).(domain.DirectChat), args.Error(1)
}
func (m *MockDirectRepo) FindByUserIDWithUsernames(ctx context.Context, userID string) ([]dto.DirectChatResponse, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]dto.DirectChatResponse), args.Error(1)
}

func startHub(t *testing.T, hub *Hub) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go hub.Run(ctx)
}

func registerClient(t *testing.T, hub *Hub, c *Client) {
	t.Helper()
	hub.register <- c
	time.Sleep(20 * time.Millisecond)
}

func makeClient(userID, username string) *Client {
	return &Client{
		userID:   userID,
		username: username,
		send:     make(chan []byte, 8),
	}
}

func TestHub_ChannelMessage_AuthoritativeFields(t *testing.T) {
	channelRepo := &MockChannelRepo{}
	messageRepo := &MockMessageRepo{}
	directRepo := &MockDirectRepo{}
	hub := NewHub(channelRepo, messageRepo, directRepo)

	sender := makeClient("user-1", "alice")
	receiver := makeClient("user-2", "bob")

	channelRepo.On("IsMember", mock.Anything, "ch-1", "user-1").Return(true, nil)
	storedAt := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	messageRepo.On("Create", mock.Anything, mock.MatchedBy(func(m domain.Message) bool {
		return m.UserID == "user-1" && m.Content == "hello" &&
			m.ChannelID != nil && *m.ChannelID == "ch-1"
	})).Return(domain.Message{ID: "msg-1", Content: "hello", CreatedAt: storedAt}, nil)
	channelRepo.On("GetMembers", mock.Anything, "ch-1").Return([]domain.User{
		{ID: "user-1"}, {ID: "user-2"},
	}, nil)

	startHub(t, hub)
	registerClient(t, hub, sender)
	registerClient(t, hub, receiver)

	raw := `{"type":"channel_message","target_id":"ch-1","content":"hello",` +
		`"sender_id":"attacker","username":"attacker"}`
	hub.broadcast <- IncomingMessage{client: sender, data: []byte(raw)}

	select {
	case got := <-receiver.send:
		var out outgoingWSMessage
		require.NoError(t, json.Unmarshal(got, &out))
		assert.Equal(t, "msg-1", out.ID)
		assert.Equal(t, "channel_message", out.Type)
		assert.Equal(t, "channel", out.TargetType)
		assert.Equal(t, "ch-1", out.TargetID)
		assert.Equal(t, "hello", out.Content)
		assert.Equal(t, "user-1", out.SenderID, "sender_id must come from JWT, not the wire")
		assert.Equal(t, "alice", out.Username, "username must come from JWT, not the wire")
		assert.Equal(t, storedAt, out.CreatedAt)
	case <-time.After(time.Second):
		t.Fatal("receiver did not get message")
	}

	select {
	case <-sender.send:
		t.Fatal("sender should not receive own message")
	default:
	}

	channelRepo.AssertExpectations(t)
	messageRepo.AssertExpectations(t)
}

func TestHub_ChannelMessage_RejectsNonMember(t *testing.T) {
	channelRepo := &MockChannelRepo{}
	messageRepo := &MockMessageRepo{}
	directRepo := &MockDirectRepo{}
	hub := NewHub(channelRepo, messageRepo, directRepo)

	intruder := makeClient("user-99", "mallory")
	target := makeClient("user-2", "bob")

	channelRepo.On("IsMember", mock.Anything, "ch-1", "user-99").Return(false, nil)

	startHub(t, hub)
	registerClient(t, hub, intruder)
	registerClient(t, hub, target)

	raw := `{"type":"channel_message","target_id":"ch-1","content":"sneaky"}`
	hub.broadcast <- IncomingMessage{client: intruder, data: []byte(raw)}

	select {
	case <-target.send:
		t.Fatal("target should not receive message from non-member")
	case <-time.After(100 * time.Millisecond):
	}

	channelRepo.AssertExpectations(t)
	messageRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestHub_DMMessage_DerivesRecipient(t *testing.T) {
	channelRepo := &MockChannelRepo{}
	messageRepo := &MockMessageRepo{}
	directRepo := &MockDirectRepo{}
	hub := NewHub(channelRepo, messageRepo, directRepo)

	sender := makeClient("user-1", "alice")
	receiver := makeClient("user-2", "bob")

	directRepo.On("FindByID", mock.Anything, "dm-1").Return(domain.DirectChat{
		ID: "dm-1", UserID1: "user-1", UserID2: "user-2",
	}, nil)
	storedAt := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	messageRepo.On("Create", mock.Anything, mock.MatchedBy(func(m domain.Message) bool {
		return m.UserID == "user-1" && m.Content == "hi" &&
			m.DirectChatID != nil && *m.DirectChatID == "dm-1"
	})).Return(domain.Message{ID: "msg-7", Content: "hi", CreatedAt: storedAt}, nil)

	startHub(t, hub)
	registerClient(t, hub, sender)
	registerClient(t, hub, receiver)

	raw := `{"type":"dm_message","target_id":"dm-1","content":"hi"}`
	hub.broadcast <- IncomingMessage{client: sender, data: []byte(raw)}

	select {
	case got := <-receiver.send:
		var out outgoingWSMessage
		require.NoError(t, json.Unmarshal(got, &out))
		assert.Equal(t, "msg-7", out.ID)
		assert.Equal(t, "dm_message", out.Type)
		assert.Equal(t, "direct", out.TargetType)
		assert.Equal(t, "dm-1", out.TargetID)
		assert.Equal(t, "user-1", out.SenderID)
		assert.Equal(t, "alice", out.Username)
	case <-time.After(time.Second):
		t.Fatal("receiver did not get DM")
	}

	select {
	case <-sender.send:
		t.Fatal("sender should not receive own DM")
	default:
	}

	directRepo.AssertExpectations(t)
	messageRepo.AssertExpectations(t)
}

func TestHub_DMMessage_RejectsNonParticipant(t *testing.T) {
	channelRepo := &MockChannelRepo{}
	messageRepo := &MockMessageRepo{}
	directRepo := &MockDirectRepo{}
	hub := NewHub(channelRepo, messageRepo, directRepo)

	intruder := makeClient("user-99", "mallory")
	victim := makeClient("user-2", "bob")

	directRepo.On("FindByID", mock.Anything, "dm-1").Return(domain.DirectChat{
		ID: "dm-1", UserID1: "user-1", UserID2: "user-2",
	}, nil)

	startHub(t, hub)
	registerClient(t, hub, intruder)
	registerClient(t, hub, victim)

	raw := `{"type":"dm_message","target_id":"dm-1","content":"i shouldn't be here"}`
	hub.broadcast <- IncomingMessage{client: intruder, data: []byte(raw)}

	select {
	case <-victim.send:
		t.Fatal("non-participant must not be able to post to a DM")
	case <-time.After(100 * time.Millisecond):
	}

	directRepo.AssertExpectations(t)
	messageRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestHub_GracefulShutdown(t *testing.T) {
	channelRepo := &MockChannelRepo{}
	messageRepo := &MockMessageRepo{}
	directRepo := &MockDirectRepo{}
	hub := NewHub(channelRepo, messageRepo, directRepo)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(done)
	}()

	c := makeClient("user-1", "alice")
	hub.register <- c
	time.Sleep(20 * time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Hub.Run did not exit after ctx cancel")
	}

	select {
	case _, ok := <-c.send:
		assert.False(t, ok, "send channel should be closed after shutdown")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("send channel should be closed, but read blocked")
	}
}
