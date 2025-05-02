package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/teamzidi/gchat-devin/devin"
	"github.com/teamzidi/gchat-devin/storage"
)

type MockDevinClient struct {
	sessions map[string]*devin.Session
	messages map[string][]*devin.Message
	usage    *devin.UsageResponse
}

func NewMockDevinClient() *MockDevinClient {
	return &MockDevinClient{
		sessions: make(map[string]*devin.Session),
		messages: make(map[string][]*devin.Message),
		usage: &devin.UsageResponse{
			RemainingCredits: 90.5,
			TotalCredits:     100.0,
		},
	}
}

func (c *MockDevinClient) CreateSession(ctx context.Context, initialMessage string) (*devin.Session, error) {
	sessionID := uuid.New().String()
	session := &devin.Session{
		ID:         sessionID,
		CreateTime: time.Now(),
		Status:     "active",
	}
	c.sessions[sessionID] = session
	
	message := &devin.Message{
		ID:         uuid.New().String(),
		Content:    initialMessage,
		Role:       "user",
		CreateTime: time.Now(),
	}
	c.messages[sessionID] = append(c.messages[sessionID], message)
	
	return session, nil
}

func (c *MockDevinClient) SendMessage(ctx context.Context, sessionID, content string) (*devin.Message, error) {
	message := &devin.Message{
		ID:         uuid.New().String(),
		Content:    content,
		Role:       "user",
		CreateTime: time.Now(),
	}
	c.messages[sessionID] = append(c.messages[sessionID], message)
	
	response := &devin.Message{
		ID:         uuid.New().String(),
		Content:    "これは " + content + " に対する応答です",
		Role:       "assistant",
		CreateTime: time.Now(),
	}
	c.messages[sessionID] = append(c.messages[sessionID], response)
	
	return message, nil
}

func (c *MockDevinClient) GetUsage(ctx context.Context) (*devin.UsageResponse, error) {
	return c.usage, nil
}

func TestHandleAddedToSpace(t *testing.T) {
	store := storage.NewMockStore()
	devinClient := NewMockDevinClient()
	
	handler := NewHandler(devinClient, store)
	
	event := &Event{
		Type: "ADDED_TO_SPACE",
		Space: &Space{
			Name:        "spaces/test-space",
			DisplayName: "Test Space",
			Type:        "ROOM",
		},
		User: &User{
			Name:        "users/test-user",
			DisplayName: "Test User",
			Email:       "test@example.com",
			Type:        "HUMAN",
		},
	}
	
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	
	handler.HandleEvent(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	
	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	
	if response.Text == "" {
		t.Error("expected non-empty response text")
	}
}

func TestHandleMessage(t *testing.T) {
	store := storage.NewMockStore()
	devinClient := NewMockDevinClient()
	
	handler := NewHandler(devinClient, store)
	
	event := &Event{
		Type: "MESSAGE",
		Message: &Message{
			Name: "spaces/test-space/messages/test-message",
			Sender: &User{
				Name:        "users/test-user",
				DisplayName: "Test User",
				Email:       "test@example.com",
				Type:        "HUMAN",
			},
			Text: "@Devin こんにちは",
			Thread: &Thread{
				Name: "spaces/test-space/threads/test-thread",
			},
			Annotations: []*Annotation{
				{
					Type:       "USER_MENTION",
					StartIndex: 0,
					Length:     6,
					Text:       "@Devin",
					UserMention: &User{
						Name:        "users/devin",
						DisplayName: "Devin",
						Type:        "BOT",
					},
				},
			},
		},
		User: &User{
			Name:        "users/test-user",
			DisplayName: "Test User",
			Email:       "test@example.com",
			Type:        "HUMAN",
		},
		Space: &Space{
			Name:        "spaces/test-space",
			DisplayName: "Test Space",
			Type:        "ROOM",
		},
	}
	
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	
	handler.HandleEvent(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	
	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	
	if response.Text == "" {
		t.Error("expected non-empty response text")
	}
}

func TestHandleUsageCommand(t *testing.T) {
	store := storage.NewMockStore()
	devinClient := NewMockDevinClient()
	
	handler := NewHandler(devinClient, store)
	
	event := &Event{
		Type: "MESSAGE",
		Message: &Message{
			Name: "spaces/test-space/messages/test-message",
			Sender: &User{
				Name:        "users/test-user",
				DisplayName: "Test User",
				Email:       "test@example.com",
				Type:        "HUMAN",
			},
			Text: "/devin usage",
			Thread: &Thread{
				Name: "spaces/test-space/threads/test-thread",
			},
		},
		User: &User{
			Name:        "users/test-user",
			DisplayName: "Test User",
			Email:       "test@example.com",
			Type:        "HUMAN",
		},
		Space: &Space{
			Name:        "spaces/test-space",
			DisplayName: "Test Space",
			Type:        "ROOM",
		},
	}
	
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	
	handler.HandleEvent(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	
	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	
	if response.Text == "" {
		t.Error("expected non-empty response text")
	}
	
	if !bytes.Contains([]byte(response.Text), []byte("使用状況")) {
		t.Errorf("expected response to contain usage information, got %q", response.Text)
	}
}

func TestContainsMention(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		annotations []*Annotation
		want        bool
	}{
		{
			name: "with mention",
			text: "@Devin こんにちは",
			annotations: []*Annotation{
				{
					Type:       "USER_MENTION",
					StartIndex: 0,
					Length:     6,
					Text:       "@Devin",
				},
			},
			want: true,
		},
		{
			name:        "without mention",
			text:        "こんにちは",
			annotations: []*Annotation{},
			want:        false,
		},
		{
			name:        "nil annotations",
			text:        "@Devin こんにちは",
			annotations: nil,
			want:        false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsMention(tt.text, tt.annotations); got != tt.want {
				t.Errorf("containsMention() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveMention(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		annotations []*Annotation
		want        string
	}{
		{
			name: "with mention",
			text: "@Devin こんにちは",
			annotations: []*Annotation{
				{
					Type:       "USER_MENTION",
					StartIndex: 0,
					Length:     6,
					Text:       "@Devin",
				},
			},
			want: "こんにちは",
		},
		{
			name:        "without mention",
			text:        "こんにちは",
			annotations: []*Annotation{},
			want:        "こんにちは",
		},
		{
			name:        "nil annotations",
			text:        "@Devin こんにちは",
			annotations: nil,
			want:        "@Devin こんにちは",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeMention(tt.text, tt.annotations); got != tt.want {
				t.Errorf("removeMention() = %q, want %q", got, tt.want)
			}
		})
	}
}
