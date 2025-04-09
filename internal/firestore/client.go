package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Client struct {
	Client     *firestore.Client
	Collection string
}

type SessionMapping struct {
	ID             string    `firestore:"-"`
	GChatSpace     string    `firestore:"gchat_space"`
	GChatThread    string    `firestore:"gchat_thread"`
	DevinSessionID string    `firestore:"devin_session_id"`
	UserID         string    `firestore:"user_id"`
	Prompt         string    `firestore:"prompt"`
	CreatedAt      time.Time `firestore:"created_at"`
	UpdatedAt      time.Time `firestore:"updated_at"`
	Messages       []Message `firestore:"messages"`
}

type Message struct {
	Sender    string    `firestore:"sender"`
	Content   string    `firestore:"content"`
	Timestamp time.Time `firestore:"timestamp"`
}

func NewClient(ctx context.Context, projectID string, collection string) (*Client, error) {
	if collection == "" {
		collection = "gchat_devin_sessions"
	}

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &Client{
		Client:     client,
		Collection: collection,
	}, nil
}

func (c *Client) CreateSessionMapping(ctx context.Context, gchatSpace, gchatThread, devinSessionID, userID, prompt string) (string, error) {
	now := time.Now()
	
	mapping := SessionMapping{
		GChatSpace:     gchatSpace,
		GChatThread:    gchatThread,
		DevinSessionID: devinSessionID,
		UserID:         userID,
		Prompt:         prompt,
		CreatedAt:      now,
		UpdatedAt:      now,
		Messages: []Message{
			{
				Sender:    "user",
				Content:   prompt,
				Timestamp: now,
			},
		},
	}

	docRef, _, err := c.Client.Collection(c.Collection).Add(ctx, mapping)
	if err != nil {
		return "", err
	}

	return docRef.ID, nil
}

func (c *Client) GetSessionByThread(ctx context.Context, gchatSpace, gchatThread string) (*SessionMapping, error) {
	query := c.Client.Collection(c.Collection).
		Where("gchat_space", "==", gchatSpace).
		Where("gchat_thread", "==", gchatThread).
		Limit(1)

	iter := query.Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, status.Errorf(codes.NotFound, "session mapping not found")
	}
	if err != nil {
		return nil, err
	}

	var mapping SessionMapping
	if err := doc.DataTo(&mapping); err != nil {
		return nil, err
	}

	mapping.ID = doc.Ref.ID
	return &mapping, nil
}

func (c *Client) AddMessage(ctx context.Context, mappingID, sender, content string) error {
	now := time.Now()
	
	docRef := c.Client.Collection(c.Collection).Doc(mappingID)
	
	_, err := docRef.Update(ctx, []firestore.Update{
		{
			Path:  "updated_at",
			Value: now,
		},
		{
			Path:  "messages",
			Value: firestore.ArrayUnion(Message{
				Sender:    sender,
				Content:   content,
				Timestamp: now,
			}),
		},
	})
	
	return err
}

func (c *Client) GetUserSessions(ctx context.Context, userID string, limit int) ([]*SessionMapping, error) {
	if limit <= 0 {
		limit = 10
	}

	query := c.Client.Collection(c.Collection).
		Where("user_id", "==", userID).
		OrderBy("created_at", firestore.Desc).
		Limit(limit)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var sessions []*SessionMapping
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var mapping SessionMapping
		if err := doc.DataTo(&mapping); err != nil {
			return nil, err
		}

		mapping.ID = doc.Ref.ID
		sessions = append(sessions, &mapping)
	}

	return sessions, nil
}

func (c *Client) Close() error {
	return c.Client.Close()
}
