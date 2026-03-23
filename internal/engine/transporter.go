package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Transporter handles HTTP communication between client and server.
type Transporter struct {
	baseURL  string
	apiKey   string
	clientID string
	client   *http.Client
}

// NewTransporter creates a new HTTP transporter for sync communication.
func NewTransporter(baseURL, apiKey, clientID string) *Transporter {
	return &Transporter{
		baseURL:  baseURL,
		apiKey:   apiKey,
		clientID: clientID,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Push sends local outbox events to the remote server.
func (t *Transporter) Push(ctx context.Context, events []SyncEvent) error {
	payload := PushPayload{
		ClientID: t.clientID,
		Events:   events,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("syncd: marshal push payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		t.baseURL+"/api/v1/sync/push", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("syncd: push request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("syncd: push failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[transporter] Pushed %d event(s) to server", len(events))
	return nil
}

// Pull fetches new remote events since the given cursor.
func (t *Transporter) Pull(ctx context.Context, cursor int64) (*PullResponse, error) {
	url := fmt.Sprintf("%s/api/v1/sync/pull?cursor=%d", t.baseURL, cursor)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("syncd: pull request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("syncd: pull failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var pullResp PullResponse
	if err := json.NewDecoder(resp.Body).Decode(&pullResp); err != nil {
		return nil, fmt.Errorf("syncd: decode pull response: %w", err)
	}

	if len(pullResp.Events) > 0 {
		log.Printf("[transporter] Pulled %d event(s) from server (cursor: %d → %d)",
			len(pullResp.Events), cursor, pullResp.Cursor)
	}

	return &pullResp, nil
}
