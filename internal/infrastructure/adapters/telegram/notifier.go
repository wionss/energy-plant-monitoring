package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Notifier struct {
	botToken string
	chatID   string
	enabled  bool
	client   *http.Client
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
	return &Notifier{
		botToken: botToken,
		chatID:   chatID,
		enabled:  enabled,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (n *Notifier) SendErrorNotification(errorType, message, context string) error {
	if !n.enabled {
		log.Printf("Telegram notifications disabled. Would have sent: %s - %s", errorType, message)
		return nil
	}

	if n.botToken == "" || n.chatID == "" {
		return fmt.Errorf("telegram bot token or chat ID not configured")
	}

	text := n.formatErrorMessage(errorType, message, context)

	return n.sendMessage(text)
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

	log.Printf("Telegram notification sent successfully")
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
