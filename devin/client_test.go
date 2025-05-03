package devin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key")
	if client.apiKey != "test-api-key" {
		t.Errorf("expected apiKey to be %q, got %q", "test-api-key", client.apiKey)
	}
	if client.httpClient == nil {
		t.Error("expected httpClient to be non-nil")
	}
}

func TestCreateSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected method to be %s, got %s", http.MethodPost, r.Method)
		}
		if r.URL.Path != "/v1/sessions" {
			t.Errorf("expected path to be %s, got %s", "/v1/sessions", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization header to be %q, got %q", "Bearer test-api-key", r.Header.Get("Authorization"))
		}

		var req CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.InitialMessage != "test message" {
			t.Errorf("expected InitialMessage to be %q, got %q", "test message", req.InitialMessage)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateSessionResponse{
			Session: Session{
				ID:         "test-session-id",
				CreateTime: time.Now(),
				Status:     "active",
			},
		})
	}))
	defer server.Close()

	client := &Client{
		apiKey: "test-api-key",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	oldBaseURL := baseURL
	baseURL = server.URL
	defer func() { baseURL = oldBaseURL }()

	session, err := client.CreateSession(context.Background(), "test message")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if session.ID != "test-session-id" {
		t.Errorf("expected session ID to be %q, got %q", "test-session-id", session.ID)
	}
	if session.Status != "active" {
		t.Errorf("expected session status to be %q, got %q", "active", session.Status)
	}
}

func TestSendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected method to be %s, got %s", http.MethodPost, r.Method)
		}
		if r.URL.Path != "/v1/sessions/test-session-id/messages" {
			t.Errorf("expected path to be %s, got %s", "/v1/sessions/test-session-id/messages", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization header to be %q, got %q", "Bearer test-api-key", r.Header.Get("Authorization"))
		}

		var req SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.Content != "test message" {
			t.Errorf("expected Content to be %q, got %q", "test message", req.Content)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(SendMessageResponse{
			Message: Message{
				ID:         "test-message-id",
				Content:    "test message",
				Role:       "user",
				CreateTime: time.Now(),
			},
		})
	}))
	defer server.Close()

	client := &Client{
		apiKey: "test-api-key",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	oldBaseURL := baseURL
	baseURL = server.URL
	defer func() { baseURL = oldBaseURL }()

	message, err := client.SendMessage(context.Background(), "test-session-id", "test message")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if message.ID != "test-message-id" {
		t.Errorf("expected message ID to be %q, got %q", "test-message-id", message.ID)
	}
	if message.Content != "test message" {
		t.Errorf("expected message content to be %q, got %q", "test message", message.Content)
	}
	if message.Role != "user" {
		t.Errorf("expected message role to be %q, got %q", "user", message.Role)
	}
}

func TestGetUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected method to be %s, got %s", http.MethodGet, r.Method)
		}
		if r.URL.Path != "/v1/usage" {
			t.Errorf("expected path to be %s, got %s", "/v1/usage", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization header to be %q, got %q", "Bearer test-api-key", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UsageResponse{
			RemainingCredits: 90.5,
			TotalCredits:     100.0,
		})
	}))
	defer server.Close()

	client := &Client{
		apiKey: "test-api-key",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	oldBaseURL := baseURL
	baseURL = server.URL
	defer func() { baseURL = oldBaseURL }()

	usage, err := client.GetUsage(context.Background())
	if err != nil {
		t.Fatalf("GetUsage failed: %v", err)
	}
	if usage.RemainingCredits != 90.5 {
		t.Errorf("expected remaining credits to be %f, got %f", 90.5, usage.RemainingCredits)
	}
	if usage.TotalCredits != 100.0 {
		t.Errorf("expected total credits to be %f, got %f", 100.0, usage.TotalCredits)
	}
}
