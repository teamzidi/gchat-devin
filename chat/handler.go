package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/teamzidi/gchat-devin/devin"
	"github.com/teamzidi/gchat-devin/storage"
)

const (
	CommandDevin = "/devin"
	CommandUsage = "usage"
)

type Handler struct {
	devinClient *devin.Client
	store       storage.Store
}

func NewHandler(devinClient *devin.Client, store storage.Store) *Handler {
	return &Handler{
		devinClient: devinClient,
		store:       store,
	}
}

func (h *Handler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("decode event: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	
	log.Printf("received event: type=%s", event.Type)
	
	switch event.Type {
	case "ADDED_TO_SPACE":
		h.handleAddedToSpace(ctx, w, &event)
	case "MESSAGE":
		h.handleMessage(ctx, w, &event)
	case "CARD_CLICKED":
		h.handleCardClicked(ctx, w, &event)
	default:
		log.Printf("unknown event type: %s", event.Type)
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) handleAddedToSpace(ctx context.Context, w http.ResponseWriter, event *Event) {
	response := &Response{
		Text: "こんにちは！Devin AI アシスタントです。`@Devin` または `/devin` でプロンプトを送信すると Devin が応答します。API 使用状況を確認するには `/devin usage` と入力してください。",
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("encode response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) handleMessage(ctx context.Context, w http.ResponseWriter, event *Event) {
	text := event.Message.Text
	
	if text == "" || !(containsMention(text, event.Message.Annotations) || strings.HasPrefix(text, CommandDevin)) {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	text = removeMention(text, event.Message.Annotations)
	
	if strings.HasPrefix(text, CommandDevin) {
		text = strings.TrimSpace(strings.TrimPrefix(text, CommandDevin))
	}
	
	if strings.HasPrefix(text, CommandUsage) {
		h.handleUsageCommand(ctx, w, event)
		return
	}
	
	if strings.TrimSpace(text) == "" {
		response := &Response{
			Text: "何かお手伝いできることはありますか？プロンプトを入力してください。",
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	
	var session *storage.Session
	var err error
	
	if event.Message.Thread != nil && event.Message.Thread.Name != "" {
		session, err = h.store.GetSessionByThreadID(ctx, event.Message.Thread.Name)
		if err != nil {
			log.Printf("get session by thread: %v", err)
		}
	}
	
	if session == nil {
		devinSession, err := h.devinClient.CreateSession(ctx, text)
		if err != nil {
			log.Printf("create devin session: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		
		threadName := event.Message.Thread.Name
		if threadName == "" {
			threadName = event.Message.Name
		}
		
		session = &storage.Session{
			ID:             uuid.New().String(),
			DevinSessionID: devinSession.ID,
			SpaceID:        event.Space.Name,
			ThreadID:       threadName,
			UserID:         event.User.Name,
			CreateTime:     time.Now(),
			UpdateTime:     time.Now(),
		}
		
		if err := h.store.CreateSession(ctx, session); err != nil {
			log.Printf("create session: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		
		userMessage := &storage.Message{
			ID:         uuid.New().String(),
			SessionID:  session.ID,
			Content:    text,
			Role:       "user",
			CreateTime: time.Now(),
		}
		
		if err := h.store.AddMessage(ctx, userMessage); err != nil {
			log.Printf("add user message: %v", err)
		}
		
		response := &Response{
			Text: "Devin が考えています...",
		}
		
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("encode processing response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		
		go h.processDevinResponseAsync(ctx, session, devinSession.ID)
		return
	}
	
	userMessage := &storage.Message{
		ID:         uuid.New().String(),
		SessionID:  session.ID,
		Content:    text,
		Role:       "user",
		CreateTime: time.Now(),
	}
	
	if err := h.store.AddMessage(ctx, userMessage); err != nil {
		log.Printf("add user message: %v", err)
	}
	
	response := &Response{
		Text: "Devin が考えています...",
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("encode processing response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	
	go h.processDevinResponseAsync(ctx, session, session.DevinSessionID)
}

func (h *Handler) handleCardClicked(ctx context.Context, w http.ResponseWriter, event *Event) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleUsageCommand(ctx context.Context, w http.ResponseWriter, event *Event) {
	usage, err := h.devinClient.GetUsage(ctx)
	if err != nil {
		log.Printf("get usage: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	
	response := &Response{
		Text: fmt.Sprintf("Devin API 使用状況:\n残りクレジット: %.2f / %.2f (%.1f%%)",
			usage.RemainingCredits,
			usage.TotalCredits,
			(usage.RemainingCredits/usage.TotalCredits)*100),
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("encode usage response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) processDevinResponseAsync(ctx context.Context, session *storage.Session, devinSessionID string) {
	
	time.Sleep(5 * time.Second)
	
	devinMessage := &devin.Message{
		ID:         uuid.New().String(),
		Content:    "これは Devin からの応答です。実際の実装では、Devin API から応答を取得します。",
		Role:       "assistant",
		CreateTime: time.Now(),
	}
	
	assistantMessage := &storage.Message{
		ID:         uuid.New().String(),
		SessionID:  session.ID,
		Content:    devinMessage.Content,
		Role:       devinMessage.Role,
		CreateTime: time.Now(),
	}
	
	const maxRetries = 3
	var attempt int
	for attempt = 1; attempt <= maxRetries; attempt++ {
		if err := h.store.AddMessage(ctx, assistantMessage); err != nil {
			log.Printf("attempt %d: failed to add assistant message: %v", attempt, err)
			if attempt < maxRetries {
				backoff := time.Duration(attempt*attempt) * time.Second
				log.Printf("retrying in %v...", backoff)
				time.Sleep(backoff)
				continue
			}
			log.Printf("max retries reached. giving up on adding assistant message.")
			return
		}
		break
	}
	
	
	chatMessage := &ChatMessage{
		Text: devinMessage.Content,
		Thread: &Thread{
			Name: session.ThreadID,
		},
	}
	
	log.Printf("sending message to chat: thread=%s", session.ThreadID)
}

func containsMention(text string, annotations []*Annotation) bool {
	if annotations == nil {
		return false
	}
	
	for _, annotation := range annotations {
		if annotation.Type == "USER_MENTION" {
			return true
		}
	}
	
	return false
}

func removeMention(text string, annotations []*Annotation) string {
	if annotations == nil {
		return text
	}
	
	for _, annotation := range annotations {
		if annotation.Type == "USER_MENTION" {
			text = strings.ReplaceAll(text, annotation.Text, "")
		}
	}
	
	return strings.TrimSpace(text)
}
