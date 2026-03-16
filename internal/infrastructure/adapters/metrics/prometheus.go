package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// EventsIngestedTotal counts the total number of events ingested from Kafka
	EventsIngestedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_ingested_total",
			Help: "Total number of events ingested from Kafka",
		},
		[]string{"event_type", "plant_id", "status"},
	)

	// EventsValidationErrors counts validation errors during event processing
	EventsValidationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_validation_errors_total",
			Help: "Total number of validation errors during event processing",
		},
		[]string{"error_type"},
	)

	// PlantStatusUpdates counts the number of plant status updates (Digital Twin)
	PlantStatusUpdates = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plant_status_updates_total",
			Help: "Total number of plant status updates",
		},
		[]string{"plant_id", "status"},
	)

	// EventProcessingDuration measures the duration of event processing
	EventProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "event_processing_duration_seconds",
			Help:    "Duration of event processing in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"event_type"},
	)

	// KafkaDLQSendsTotal counts the total number of messages sent to the DLQ
	KafkaDLQSendsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_dlq_sends_total",
			Help: "Messages sent to DLQ",
		},
		[]string{"topic", "reason"},
	)

	// KafkaRetryAttemptsTotal counts the total number of Kafka retry attempts
	KafkaRetryAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_retry_attempts_total",
			Help: "Kafka retry attempts",
		},
		[]string{"topic"},
	)
)
