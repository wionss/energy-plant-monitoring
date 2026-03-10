package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/sony/gobreaker"
)

type Adapter struct {
	client *http.Client
	cb     *gobreaker.CircuitBreaker
}

var _ output.WebhookAdapterInterface = &Adapter{}

func NewAdapter(client *http.Client) *Adapter {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "webhook",
		MaxRequests: 1,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("webhook circuit breaker state change",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	})
	return &Adapter{client: client, cb: cb}
}

func (a *Adapter) SendPayload(url string, payload any) error {
	_, err := a.cb.Execute(func() (any, error) {
		return nil, a.doSend(url, payload)
	})
	return err
}

func (a *Adapter) doSend(url string, payload any) error {
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
