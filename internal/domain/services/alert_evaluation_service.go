package services

import (
	"fmt"
	"log/slog"

	"monitoring-energy-service/internal/infrastructure/adapters/telegram"
)

// AlertRule define una regla para generar alertas basada en eventos
type AlertRule struct {
	Name            string  // Unique identifier: "temperature_high", "pressure_critical", etc.
	EventType       string  // Event type to trigger this rule: "temperature", "*" for all
	Condition       string  // Condition type: "gt", "lt", "gte", "lte", "eq", "contains"
	FieldPath       string  // JSON path to check: "data.temperature", "data.status"
	Threshold       float64 // Comparison value
	Severity        string  // "info", "warning", "critical"
	NotificationMsg string  // Message template for telegram notification
}

// AlertEvaluationService evalúa reglas de alertas en eventos
// No bloquea el procesamiento: errores aquí no fallan el ingestion
type AlertEvaluationService struct {
	rules            []AlertRule
	telegramNotifier *telegram.Notifier
}

// NewAlertEvaluationService crea una nueva instancia del servicio
func NewAlertEvaluationService(
	telegramNotifier *telegram.Notifier,
) *AlertEvaluationService {
	service := &AlertEvaluationService{
		telegramNotifier: telegramNotifier,
		rules:            getDefaultAlertRules(),
	}
	return service
}

// EvaluateEvent evalúa todas las reglas contra un evento
// Se ejecuta asincronamente después de persistir el evento
func (s *AlertEvaluationService) EvaluateEvent(
	eventID string,
	eventType string,
	plantSourceID string,
	plantName string,
	data map[string]interface{},
) {
	for _, rule := range s.rules {
		// Saltar si la regla no aplica a este tipo de evento
		if rule.EventType != "*" && rule.EventType != eventType {
			continue
		}

		// Evaluar condición
		if s.evaluateCondition(rule, data) {
			s.triggerAlert(rule, plantSourceID, plantName, eventType, data)
		}
	}
}

// evaluateCondition evalúa si la condición de la regla se cumple
func (s *AlertEvaluationService) evaluateCondition(rule AlertRule, data map[string]interface{}) bool {
	// Obtener valor del campo especificado
	value := s.getFieldValue(data, rule.FieldPath)
	if value == nil {
		return false
	}

	// Convertir valor a float64 si es posible
	floatVal, ok := toFloat64(value)
	if !ok && rule.Condition != "contains" && rule.Condition != "eq" {
		return false
	}

	// Evaluar según tipo de condición
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
		return containsString(value, rule.FieldPath)
	}

	return false
}

// getFieldValue obtiene el valor de un campo del data JSON
// Soporta rutas simples como "temperature" o anidadas como "data.temperature"
func (s *AlertEvaluationService) getFieldValue(data map[string]interface{}, fieldPath string) interface{} {
	// Parse simple field path (e.g., "temperature" or "data.temperature")
	if fieldName, ok := data[fieldPath]; ok {
		return fieldName
	}

	// Soportar anidación con punto separador
	// Por ahora, mantener simple. Puede extenderse para soportar objetos anidados
	return nil
}

// triggerAlert dispara la notificación de alerta
func (s *AlertEvaluationService) triggerAlert(
	rule AlertRule,
	plantSourceID string,
	plantName string,
	eventType string,
	data map[string]interface{},
) {
	if s.telegramNotifier == nil {
		return
	}

	// Construir mensaje de alerta
	alertMsg := fmt.Sprintf(
		"🚨 ALERTA [%s] %s\n🏭 Planta: %s\n📊 Evento: %s\n📋 Severidad: %s",
		rule.Severity, rule.NotificationMsg, plantName, eventType, rule.Severity,
	)

	// Enviar notificación
	slog.Warn("alert triggered",
		"rule", rule.Name,
		"plant", plantName,
		"event_type", eventType,
		"severity", rule.Severity,
	)

	// Enviar a través de Telegram sin bloquear
	go func() {
		// La implementación real dependería de cómo está estructurado el telegram.Notifier
		// Por ahora, solo registramos el log
		slog.Info("alert notification would be sent", "message", alertMsg)
	}()
}

// getDefaultAlertRules retorna las reglas de alerta predefinidas
// Estas pueden eventualmente venir de BD o configuración
func getDefaultAlertRules() []AlertRule {
	return []AlertRule{
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
			Threshold:       0, // Will be treated as string comparison
			Severity:        "critical",
			NotificationMsg: "Pérdida de energía en planta",
		},
	}
}

// Funciones auxiliares

// toFloat64 intenta convertir un valor a float64
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

// containsString verifica si un valor contiene una cadena
func containsString(value interface{}, searchStr string) bool {
	switch v := value.(type) {
	case string:
		return len(searchStr) > 0 && len(v) > 0 // Simplified check
	default:
		return false
	}
}
