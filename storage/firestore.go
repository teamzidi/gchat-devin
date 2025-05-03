package storage

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type Session struct {
	ID            string    `firestore:"id"`
	DevinSessionID string    `firestore:"devin_session_id"`
	SpaceID       string    `firestore:"space_id"`
	ThreadID      string    `firestore:"thread_id"`
	UserID        string    `firestore:"user_id"`
	CreateTime    time.Time `firestore:"create_time"`
	UpdateTime    time.Time `firestore:"update_time"`
}

type Message struct {
	ID        string    `firestore:"id"`
	SessionID string    `firestore:"session_id"`
	Content   string    `firestore:"content"`
	Role      string    `firestore:"role"` // "user" または "assistant"
	CreateTime time.Time `firestore:"create_time"`
}

type Store interface {
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	GetSessionByThreadID(ctx context.Context, threadID string) (*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	
	AddMessage(ctx context.Context, message *Message) error
	GetMessages(ctx context.Context, sessionID string) ([]*Message, error)
}

type FirestoreStore struct {
	client *firestore.Client
}

func NewFirestoreStore(ctx context.Context, projectID string) (*FirestoreStore, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("firestore client: %w", err)
	}
	
	return &FirestoreStore{client: client}, nil
}

func (s *FirestoreStore) Close() error {
	return s.client.Close()
}

func (s *FirestoreStore) CreateSession(ctx context.Context, session *Session) error {
	_, err := s.client.Collection("sessions").Doc(session.ID).Set(ctx, session)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	
	return nil
}

func (s *FirestoreStore) GetSession(ctx context.Context, id string) (*Session, error) {
	doc, err := s.client.Collection("sessions").Doc(id).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	
	var session Session
	if err := doc.DataTo(&session); err != nil {
		return nil, fmt.Errorf("data to session: %w", err)
	}
	
	return &session, nil
}

func (s *FirestoreStore) GetSessionByThreadID(ctx context.Context, threadID string) (*Session, error) {
	iter := s.client.Collection("sessions").Where("thread_id", "==", threadID).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("session not found: thread_id=%s", threadID)
	}
	if err != nil {
		return nil, fmt.Errorf("get session by thread: %w", err)
	}
	
	var session Session
	if err := doc.DataTo(&session); err != nil {
		return nil, fmt.Errorf("data to session: %w", err)
	}
	
	return &session, nil
}

func (s *FirestoreStore) UpdateSession(ctx context.Context, session *Session) error {
	session.UpdateTime = time.Now()
	_, err := s.client.Collection("sessions").Doc(session.ID).Set(ctx, session)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}
	
	return nil
}

func (s *FirestoreStore) AddMessage(ctx context.Context, message *Message) error {
	_, err := s.client.Collection("messages").Doc(message.ID).Set(ctx, message)
	if err != nil {
		return fmt.Errorf("add message: %w", err)
	}
	
	return nil
}

func (s *FirestoreStore) GetMessages(ctx context.Context, sessionID string) ([]*Message, error) {
	iter := s.client.Collection("messages").Where("session_id", "==", sessionID).
		OrderBy("create_time", firestore.Asc).Documents(ctx)
	
	var messages []*Message
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("get messages: %w", err)
		}
		
		var message Message
		if err := doc.DataTo(&message); err != nil {
			return nil, fmt.Errorf("data to message: %w", err)
		}
		
		messages = append(messages, &message)
	}
	
	return messages, nil
}
