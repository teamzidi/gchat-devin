package devin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL = "https://api.devin.ai"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type Session struct {
	ID        string    `json:"id"`
	CreateTime time.Time `json:"createTime"`
	Status    string    `json:"status"`
}

type Message struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Role      string    `json:"role"`
	CreateTime time.Time `json:"createTime"`
}

type CreateSessionRequest struct {
	InitialMessage string `json:"initialMessage"`
}

type CreateSessionResponse struct {
	Session Session `json:"session"`
}

type SendMessageRequest struct {
	Content string `json:"content"`
}

type SendMessageResponse struct {
	Message Message `json:"message"`
}

type UsageResponse struct {
	RemainingCredits float64 `json:"remainingCredits"`
	TotalCredits     float64 `json:"totalCredits"`
}

func (c *Client) CreateSession(ctx context.Context, initialMessage string) (*Session, error) {
	reqBody := CreateSessionRequest{
		InitialMessage: initialMessage,
	}
	
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/sessions", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status=%d body=%s", resp.StatusCode, body)
	}
	
	var createResp CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	
	return &createResp.Session, nil
}

func (c *Client) SendMessage(ctx context.Context, sessionID, content string) (*Message, error) {
	reqBody := SendMessageRequest{
		Content: content,
	}
	
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, 
		fmt.Sprintf("%s/v1/sessions/%s/messages", baseURL, sessionID), bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status=%d body=%s", resp.StatusCode, body)
	}
	
	var sendResp SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&sendResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	
	return &sendResp.Message, nil
}

func (c *Client) GetUsage(ctx context.Context) (*UsageResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/usage", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status=%d body=%s", resp.StatusCode, body)
	}
	
	var usage UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	
	return &usage, nil
}
