package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/teamzidi/gchat-devin/internal/devinapi"
	"github.com/teamzidi/gchat-devin/internal/firestore"
	"github.com/teamzidi/gchat-devin/internal/secretmanager"
)

var (
	projectID          = os.Getenv("GOOGLE_CLOUD_PROJECT")
	devinAPIKeySecret  = getEnvWithDefault("DEVIN_API_KEY_SECRET", "devin-api-key")
	port               = getEnvWithDefault("PORT", "8080")
	firestoreCollection = getEnvWithDefault("FIRESTORE_COLLECTION", "gchat_devin_sessions")
)

var (
	secretManagerClient *secretmanager.Client
	devinAPIClient      *devinapi.Client
	firestoreClient     *firestore.Client
)

type ChatEvent struct {
	Type    string                 `json:"type"`
	Message *ChatMessage           `json:"message,omitempty"`
	Space   map[string]interface{} `json:"space"`
	User    map[string]interface{} `json:"user"`
	Thread  *ChatThread            `json:"thread,omitempty"`
}

type ChatMessage struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

type ChatThread struct {
	Name string `json:"name"`
}

type ChatResponse struct {
	Text string `json:"text"`
}

func main() {
	ctx := context.Background()

	var err error
	secretManagerClient, err = secretmanager.NewClient(projectID)
	if err != nil {
		log.Fatalf("Failed to create Secret Manager client: %v", err)
	}
	defer secretManagerClient.Close()

	devinAPIKey, err := secretManagerClient.GetSecret(ctx, devinAPIKeySecret, "latest")
	if err != nil {
		log.Fatalf("Failed to get Devin API key: %v", err)
	}

	devinAPIClient = devinapi.NewClient(devinAPIKey, "")

	firestoreClient, err = firestore.NewClient(ctx, projectID, firestoreCollection)
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	http.HandleFunc("/", handleWebhook)
	http.HandleFunc("/health", handleHealthCheck)

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event ChatEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Received event: %s", event.Type)

	if event.Type != "MESSAGE" {
		respondWithJSON(w, ChatResponse{Text: "Event type not supported"})
		return
	}

	if event.Message == nil {
		respondWithJSON(w, ChatResponse{Text: "No message in event"})
		return
	}

	messageText := event.Message.Text
	spaceID, _ := event.Space["name"].(string)
	userID, _ := event.User["name"].(string)
	var threadID string
	if event.Thread != nil {
		threadID = event.Thread.Name
	} else {
		threadID = event.Message.Name
	}

	isMention := strings.Contains(messageText, "@Devin")

	isCommand := strings.HasPrefix(strings.TrimSpace(messageText), "/devin")

	if (isMention || isCommand) && strings.Contains(messageText, "API") && 
	   (strings.Contains(messageText, "usage") || strings.Contains(messageText, "残り")) {
		handleAPIUsageCheck(w, r.Context(), spaceID, threadID)
		return
	}

	if isMention || isCommand {
		cleanMessage := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(messageText, "@Devin", ""), "/devin", ""))

		sessionMapping, err := firestoreClient.GetSessionByThread(r.Context(), spaceID, threadID)
		if err == nil && sessionMapping != nil {
			handleExistingSession(w, r.Context(), sessionMapping, cleanMessage, spaceID, threadID)
		} else {
			handleNewSession(w, r.Context(), cleanMessage, spaceID, threadID, userID)
		}
		return
	}

	respondWithJSON(w, ChatResponse{Text: ""})
}

func handleAPIUsageCheck(w http.ResponseWriter, ctx context.Context, spaceID, threadID string) {
	consumptionData, err := devinAPIClient.GetEnterpriseConsumption()
	if err != nil {
		log.Printf("Failed to get enterprise consumption data: %v", err)
		respondWithJSON(w, ChatResponse{Text: "APIの使用状況を取得できませんでした。エラーが発生しました。"})
		return
	}

	usageText := "# Devin API Usage\n\n"
	usageText += fmt.Sprintf("残りクレジット: %d\n", consumptionData.RemainingCredits)
	usageText += fmt.Sprintf("合計クレジット: %d\n", consumptionData.TotalCredits)

	if len(consumptionData.UsageByDay) > 0 {
		usageText += "\n## 日別使用量\n\n"
		for _, dayUsage := range consumptionData.UsageByDay {
			usageText += fmt.Sprintf("- %s: %d credits\n", dayUsage.Date, dayUsage.Credits)
		}
	}

	respondWithJSON(w, ChatResponse{Text: usageText})
}

func handleExistingSession(w http.ResponseWriter, ctx context.Context, sessionMapping *firestore.SessionMapping, message, spaceID, threadID string) {
	devinSessionID := sessionMapping.DevinSessionID
	mappingID := sessionMapping.ID

	_, err := devinAPIClient.SendMessage(devinSessionID, message)
	if err != nil {
		log.Printf("Failed to send message to Devin: %v", err)
		respondWithJSON(w, ChatResponse{Text: "Devin にメッセージを送信できませんでした。エラーが発生しました。"})
		return
	}

	err = firestoreClient.AddMessage(ctx, mappingID, "user", message)
	if err != nil {
		log.Printf("Failed to add message to Firestore: %v", err)
	}

	sessionDetails, err := devinAPIClient.GetSessionDetails(devinSessionID)
	if err != nil {
		log.Printf("Failed to get session details: %v", err)
		respondWithJSON(w, ChatResponse{Text: "セッション詳細を取得できませんでした。エラーが発生しました。"})
		return
	}

	responseText := fmt.Sprintf(
		"メッセージを Devin に送信しました。Devin は応答を処理しています。\n\n"+
			"セッション URL: %s",
		sessionDetails.URL,
	)

	respondWithJSON(w, ChatResponse{Text: responseText})
}

func handleNewSession(w http.ResponseWriter, ctx context.Context, prompt, spaceID, threadID, userID string) {
	session, err := devinAPIClient.CreateSession(prompt)
	if err != nil {
		log.Printf("Failed to create Devin session: %v", err)
		respondWithJSON(w, ChatResponse{Text: "Devin セッションを作成できませんでした。エラーが発生しました。"})
		return
	}

	if session.SessionID == "" {
		log.Printf("No session ID returned from Devin API")
		respondWithJSON(w, ChatResponse{Text: "Devin セッションを作成できませんでした。セッション ID が返されませんでした。"})
		return
	}

	_, err = firestoreClient.CreateSessionMapping(ctx, spaceID, threadID, session.SessionID, userID, prompt)
	if err != nil {
		log.Printf("Failed to create session mapping in Firestore: %v", err)
	}

	responseText := fmt.Sprintf(
		"新しい Devin セッションを作成しました。\n\n"+
			"プロンプト: %s\n\n"+
			"セッション URL: %s\n\n"+
			"このスレッドで会話を続けることができます。",
		prompt, session.URL,
	)

	respondWithJSON(w, ChatResponse{Text: responseText})
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, map[string]string{"status": "healthy"})
}

func respondWithJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
