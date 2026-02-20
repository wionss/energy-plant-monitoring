package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type alertMessage struct {
	text string
}

type Notifier struct {
	botToken  string
	chatID    string
	enabled   bool
	client    *http.Client
	alertChan chan alertMessage
	stopOnce  sync.Once
	stopMu    sync.Mutex
	stopped   bool
	wg        sync.WaitGroup
}

type sendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type telegramResponse struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

func NewNotifier(botToken, chatID string, enabled bool) *Notifier {
	n := &Notifier{
		botToken:  botToken,
		chatID:    chatID,
		enabled:   enabled,
		client:    &http.Client{Timeout: 10 * time.Second},
		alertChan: make(chan alertMessage, 100),
	}

	n.wg.Add(1)
	go n.worker()

	return n
}

func (n *Notifier) worker() {
	defer n.wg.Done()

	for msg := range n.alertChan {
		if err := n.sendMessage(msg.text); err != nil {
			slog.Error("failed to send Telegram notification", "error", err)
		} else {
			slog.Info("Telegram notification sent")
		}
		// Simple rate limit: 1 message per 100ms
		time.Sleep(100 * time.Millisecond)
	}
}

func (n *Notifier) enqueue(text string) {
	n.stopMu.Lock()
	defer n.stopMu.Unlock()

	if n.stopped {
		slog.Warn("Telegram notifier stopped, dropping message")
		return
	}

	select {
	case n.alertChan <- alertMessage{text: text}:
	default:
		slog.Warn("Telegram alert channel full, dropping message")
	}
}

func (n *Notifier) Stop() {
	n.stopOnce.Do(func() {
		slog.Info("stopping Telegram notifier")

		n.stopMu.Lock()
		n.stopped = true
		n.stopMu.Unlock()

		close(n.alertChan)
		n.wg.Wait()
		slog.Info("Telegram notifier stopped")
	})
}

func (n *Notifier) SendErrorNotification(errorType, message, context string) error {
	if !n.enabled {
		slog.Info("Telegram notifications disabled", "error_type", errorType, "message", message)
		return nil
	}

	if n.botToken == "" || n.chatID == "" {
		return fmt.Errorf("telegram bot token or chat ID not configured")
	}

	text := n.formatErrorMessage(errorType, message, context)
	n.enqueue(text)
	return nil
}

func (n *Notifier) formatErrorMessage(errorType, message, context string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	msg := fmt.Sprintf("🚨 <b>Error de Validación</b>\n\n")
	msg += fmt.Sprintf("⏰ <b>Hora:</b> %s\n", escapeHTML(timestamp))
	msg += fmt.Sprintf("🔴 <b>Tipo:</b> %s\n", escapeHTML(errorType))
	msg += fmt.Sprintf("📝 <b>Mensaje:</b> %s\n", escapeHTML(message))

	if context != "" {
		msg += fmt.Sprintf("📋 <b>Contexto:</b> %s\n", escapeHTML(context))
	}

	msg += fmt.Sprintf("\n🏢 <b>Servicio:</b> Monitoring Energy Service")

	return msg
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func (n *Notifier) sendMessage(text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)

	reqBody := sendMessageRequest{
		ChatID:    n.chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}

	resp, err := n.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error sending telegram message: %w", err)
	}
	defer resp.Body.Close()

	var telegramResp telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return fmt.Errorf("error decoding telegram response: %w", err)
	}

	if !telegramResp.Ok {
		return fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	return nil
}

func (n *Notifier) SendValidationError(fieldName, value, errorDetail string) error {
	context := fmt.Sprintf("Campo: %s, Valor: %s", fieldName, value)
	return n.SendErrorNotification("Validación de Datos", errorDetail, context)
}

func (n *Notifier) SendUUIDError(fieldName, invalidUUID, errorDetail string) error {
	context := fmt.Sprintf("Campo: %s, UUID inválido: %s", fieldName, invalidUUID)
	return n.SendErrorNotification("UUID Inválido", errorDetail, context)
}
