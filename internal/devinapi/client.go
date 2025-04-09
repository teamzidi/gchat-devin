package devinapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

func NewClient(apiKey string, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.devin.ai/v1"
	}
	return &Client{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

type SessionResponse struct {
	SessionID    string `json:"session_id"`
	URL          string `json:"url"`
	IsNewSession bool   `json:"is_new_session"`
}

type MessageResponse struct {
	Status string `json:"status"`
}

type ConsumptionData struct {
	RemainingCredits int `json:"remaining_credits"`
	TotalCredits     int `json:"total_credits"`
	UsageByDay       []struct {
		Date    string `json:"date"`
		Credits int    `json:"credits"`
	} `json:"usage_by_day"`
}

func (c *Client) CreateSession(prompt string) (*SessionResponse, error) {
	url := fmt.Sprintf("%s/sessions", c.BaseURL)
	payload := map[string]interface{}{
		"prompt": prompt,
	}

	resp, err := c.sendRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sessionResp SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &sessionResp, nil
}

func (c *Client) SendMessage(sessionID, message string) (*MessageResponse, error) {
	url := fmt.Sprintf("%s/session/%s/message", c.BaseURL, sessionID)
	payload := map[string]interface{}{
		"message": message,
	}

	resp, err := c.sendRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var messageResp MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&messageResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &messageResp, nil
}

func (c *Client) GetSessionDetails(sessionID string) (*SessionResponse, error) {
	url := fmt.Sprintf("%s/session/%s", c.BaseURL, sessionID)

	resp, err := c.sendRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sessionResp SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &sessionResp, nil
}

func (c *Client) GetEnterpriseConsumption() (*ConsumptionData, error) {
	url := fmt.Sprintf("%s/enterprise/consumption", c.BaseURL)

	resp, err := c.sendRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var consumptionData ConsumptionData
	if err := json.NewDecoder(resp.Body).Decode(&consumptionData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &consumptionData, nil
}

func (c *Client) sendRequest(method, url string, payload interface{}) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}
		body = bytes.NewBuffer(payloadBytes)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}
