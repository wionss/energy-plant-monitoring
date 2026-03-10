package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"monitoring-energy-service/internal/domain/entities"
	"monitoring-energy-service/internal/domain/ports/output"
	"monitoring-energy-service/internal/infrastructure/adapters/telegram"

	"github.com/tidwall/gjson"
)

// AlertEvaluationService evaluates alert rules against incoming events.
// It does not block event ingestion: errors here never fail the ingestion path.
type AlertEvaluationService struct {
	mu               sync.RWMutex
	rules            []entities.AlertRule
	telegramNotifier *telegram.Notifier
	rulesRepo        output.AlertRulesRepositoryInterface
	stopCh           chan struct{}
}

// NewAlertEvaluationService creates a new instance of AlertEvaluationService.
// It loads rules from the DB on startup, falling back to hardcoded defaults on error.
func NewAlertEvaluationService(
	telegramNotifier *telegram.Notifier,
	rulesRepo output.AlertRulesRepositoryInterface,
) *AlertEvaluationService {
	svc := &AlertEvaluationService{
		telegramNotifier: telegramNotifier,
		rulesRepo:        rulesRepo,
		stopCh:           make(chan struct{}),
	}
	if rules, err := rulesRepo.FindActive(); err == nil {
		svc.rules = rules
	} else {
		svc.rules = defaultAlertRules()
		slog.Warn("failed to load alert rules from DB, using defaults", "error", err)
	}
	return svc
}

// Start launches the hot-reload goroutine that refreshes rules from the DB every 30s.
func (s *AlertEvaluationService) Start() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				if rules, err := s.rulesRepo.FindActive(); err == nil {
					s.mu.Lock()
					s.rules = rules
					s.mu.Unlock()
					slog.Debug("alert rules reloaded", "count", len(rules))
				}
			}
		}
	}()
}

// Stop signals the hot-reload goroutine to exit.
func (s *AlertEvaluationService) Stop() {
	close(s.stopCh)
}

// EvaluateEvent evaluates all active rules against an event.
// Called asynchronously after event persistence.
func (s *AlertEvaluationService) EvaluateEvent(
	eventID string,
	eventType string,
	plantSourceID string,
	plantName string,
	data map[string]interface{},
) {
	s.mu.RLock()
	rules := s.rules
	s.mu.RUnlock()

	for _, rule := range rules {
		if rule.EventType != "*" && rule.EventType != eventType {
			continue
		}
		if s.evaluateCondition(rule, data) {
			s.triggerAlert(rule, plantSourceID, plantName, eventType, data)
		}
	}
}

// evaluateCondition checks whether a rule's condition is satisfied by the event data.
func (s *AlertEvaluationService) evaluateCondition(rule entities.AlertRule, data map[string]interface{}) bool {
	value := s.getFieldValue(data, rule.FieldPath)
	if value == nil {
		return false
	}

	floatVal, ok := toFloat64(value)
	if !ok && rule.Condition != "contains" && rule.Condition != "eq" {
		return false
	}

	switch rule.Condition {
	case "gt":
		return floatVal > rule.Threshold
	case "gte":
		return floatVal >= rule.Threshold
	case "lt":
		return floatVal < rule.Threshold
	case "lte":
		return floatVal <= rule.Threshold
	case "eq":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", rule.Threshold)
	case "contains":
		return containsString(value, rule.ThresholdStr)
	}

	return false
}

// getFieldValue extracts a field value from event data using a JSONPath expression.
// Falls back to direct key lookup for backward compatibility with flat keys.
func (s *AlertEvaluationService) getFieldValue(data map[string]interface{}, fieldPath string) interface{} {
	// 1. Direct key lookup (backward compat for flat keys like "temperature")
	if v, ok := data[fieldPath]; ok {
		return v
	}
	// 2. JSONPath via gjson: supports "data.temperature", "sensors.0.value", etc.
	b, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	result := gjson.GetBytes(b, fieldPath)
	if !result.Exists() {
		return nil
	}
	return result.Value()
}

// triggerAlert sends a Telegram notification for the triggered rule.
func (s *AlertEvaluationService) triggerAlert(
	rule entities.AlertRule,
	plantSourceID string,
	plantName string,
	eventType string,
	data map[string]interface{},
) {
	if s.telegramNotifier == nil {
		return
	}

	alertMsg := fmt.Sprintf(
		"🚨 ALERTA [%s] %s\n🏭 Planta: %s\n📊 Evento: %s\n📋 Severidad: %s",
		rule.Severity, rule.NotificationMsg, plantName, eventType, rule.Severity,
	)

	slog.Warn("alert triggered",
		"rule", rule.Name,
		"plant", plantName,
		"event_type", eventType,
		"severity", rule.Severity,
	)

	if err := s.telegramNotifier.SendErrorNotification(rule.Severity, rule.NotificationMsg, alertMsg); err != nil {
		slog.Error("failed to send alert notification", "rule", rule.Name, "error", err)
	}
}

// defaultAlertRules returns hardcoded fallback rules used when the DB is unavailable.
func defaultAlertRules() []entities.AlertRule {
	return []entities.AlertRule{
		{
			Name:            "high_temperature",
			EventType:       "temperature",
			Condition:       "gt",
			FieldPath:       "temperature",
			Threshold:       50,
			Severity:        "warning",
			NotificationMsg: "Temperatura muy alta detectada",
		},
		{
			Name:            "critical_temperature",
			EventType:       "temperature",
			Condition:       "gt",
			FieldPath:       "temperature",
			Threshold:       80,
			Severity:        "critical",
			NotificationMsg: "¡Temperatura crítica! Intervención inmediata requerida",
		},
		{
			Name:            "pressure_critical",
			EventType:       "pressure",
			Condition:       "gt",
			FieldPath:       "pressure",
			Threshold:       100,
			Severity:        "critical",
			NotificationMsg: "Presión crítica detectada",
		},
		{
			Name:            "power_loss",
			EventType:       "status",
			Condition:       "eq",
			FieldPath:       "status",
			Threshold:       0,
			Severity:        "critical",
			NotificationMsg: "Pérdida de energía en planta",
		},
	}
}

// toFloat64 attempts to convert a value to float64.
func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		var f float64
		_, err := fmt.Sscanf(v, "%f", &f)
		return f, err == nil
	default:
		return 0, false
	}
}

// containsString checks whether a value contains the given substring.
func containsString(value interface{}, searchStr string) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	return strings.Contains(s, searchStr)
}
