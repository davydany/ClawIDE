package mcpserve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type notificationRequest struct {
	Title     string `json:"title"`
	Body      string `json:"body,omitempty"`
	Source    string `json:"source,omitempty"`
	Level     string `json:"level,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	FeatureID string `json:"feature_id,omitempty"`
	PaneID    string `json:"pane_id,omitempty"`
	CWD       string `json:"cwd,omitempty"`
}

func NewClient() *Client {
	baseURL := os.Getenv("CLAWIDE_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9800"
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) PostNotification(req notificationRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/notifications",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("posting notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("notification API returned status %d", resp.StatusCode)
	}

	return nil
}
