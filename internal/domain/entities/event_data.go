package entities

import "github.com/go-playground/validator/v10"

// EventData represents the validated data payload from energy monitoring events
type EventData struct {
	PowerGeneratedMW   *float64 `json:"power_generated_mw" validate:"omitempty,gte=0"`
	PowerConsumedMW    *float64 `json:"power_consumed_mw" validate:"omitempty,gte=0"`
	EfficiencyPercent  *float64 `json:"efficiency_percent" validate:"omitempty,gte=0,lte=100"`
	TemperatureCelsius *float64 `json:"temperature_celsius"`
}

var validate = validator.New()

// Validate validates the EventData fields using struct tags
func (e *EventData) Validate() error {
	return validate.Struct(e)
}
