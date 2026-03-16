package entities

// AlertRule defines a rule for generating alerts based on events.
type AlertRule struct {
	Name            string
	EventType       string
	Condition       string
	FieldPath       string
	Threshold       float64
	ThresholdStr    string
	Severity        string
	NotificationMsg string
}
