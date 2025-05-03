package storage

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

type MockStore struct {
	sessions map[string]*Session
	messages map[string][]*Message
}

func NewMockStore() *MockStore {
	return &MockStore{
		sessions: make(map[string]*Session),
		messages: make(map[string][]*Message),
	}
}

func (s *MockStore) CreateSession(ctx context.Context, session *Session) error {
	s.sessions[session.ID] = session
	return nil
}

func (s *MockStore) GetSession(ctx context.Context, id string) (*Session, error) {
	session, ok := s.sessions[id]
	if !ok {
		return nil, firestore.ErrNotFound
	}
	return session, nil
}

func (s *MockStore) GetSessionByThreadID(ctx context.Context, threadID string) (*Session, error) {
	for _, session := range s.sessions {
		if session.ThreadID == threadID {
			return session, nil
		}
	}
	return nil, firestore.ErrNotFound
}

func (s *MockStore) UpdateSession(ctx context.Context, session *Session) error {
	s.sessions[session.ID] = session
	return nil
}

func (s *MockStore) AddMessage(ctx context.Context, message *Message) error {
	s.messages[message.SessionID] = append(s.messages[message.SessionID], message)
	return nil
}

func (s *MockStore) GetMessages(ctx context.Context, sessionID string) ([]*Message, error) {
	return s.messages[sessionID], nil
}

func TestMockStore(t *testing.T) {
	ctx := context.Background()
	store := NewMockStore()

	session := &Session{
		ID:             "test-session-id",
		DevinSessionID: "devin-session-id",
		SpaceID:        "space-id",
		ThreadID:       "thread-id",
		UserID:         "user-id",
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
	}

	if err := store.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	retrievedSession, err := store.GetSession(ctx, "test-session-id")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if retrievedSession.ID != session.ID {
		t.Errorf("expected session ID to be %q, got %q", session.ID, retrievedSession.ID)
	}

	threadSession, err := store.GetSessionByThreadID(ctx, "thread-id")
	if err != nil {
		t.Fatalf("GetSessionByThreadID failed: %v", err)
	}
	if threadSession.ID != session.ID {
		t.Errorf("expected session ID to be %q, got %q", session.ID, threadSession.ID)
	}

	session.UserID = "updated-user-id"
	if err := store.UpdateSession(ctx, session); err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}

	updatedSession, err := store.GetSession(ctx, "test-session-id")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if updatedSession.UserID != "updated-user-id" {
		t.Errorf("expected user ID to be %q, got %q", "updated-user-id", updatedSession.UserID)
	}

	message := &Message{
		ID:         "test-message-id",
		SessionID:  "test-session-id",
		Content:    "test message",
		Role:       "user",
		CreateTime: time.Now(),
	}
	if err := store.AddMessage(ctx, message); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	messages, err := store.GetMessages(ctx, "test-session-id")
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].ID != "test-message-id" {
		t.Errorf("expected message ID to be %q, got %q", "test-message-id", messages[0].ID)
	}
}
