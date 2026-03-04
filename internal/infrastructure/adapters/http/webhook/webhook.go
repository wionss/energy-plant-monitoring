package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"monitoring-energy-service/internal/domain/ports/output"
)

type Adapter struct {
	client *http.Client
}

var _ output.WebhookAdapterInterface = &Adapter{}

func NewAdapter(client *http.Client) *Adapter {
	return &Adapter{
		client: client,
	}
}

func (a *Adapter) SendPayload(url string, payload any) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	slog.Info("sending webhook request", "url", url)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending webhook request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck // drain to allow connection reuse

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	return nil
}
