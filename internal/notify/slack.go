package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type SlackMessage struct {
	Text string `json:"text"`
}


func SendSlackNotification(webhook, msg string) error {
	if webhook == "" {
		return errors.New("slack webhook URL is not configured")
	}
	body, err := json.Marshal(SlackMessage{Text: msg})
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(webhook, "application/json", bytes.NewBuffer(body))

	if err != nil {
		return fmt.Errorf("failed to send slack request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned non-2xx status: %s", resp.Status)
	}

	return nil
}
