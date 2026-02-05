package api

import (
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"monitoring-energy-service/internal/domain/ports/input"

	"github.com/google/uuid"
)

type EventGenerator struct {
	kafkaService input.KafkaServiceInterface
	topic        string
	stopChan     chan struct{}
}

type EnergyMonitoringEvent struct {
	PlantID        string    `json:"plant_id"`
	PlantSourceId  uuid.UUID `json:"plant_source_id"`
	PlantName      string    `json:"plant_name"`
	EventType      string    `json:"event_type"`
	PowerGenerated float64   `json:"power_generated_mw"`
	PowerConsumed  float64   `json:"power_consumed_mw"`
	Efficiency     float64   `json:"efficiency_percent"`
	Temperature    float64   `json:"temperature_celsius"`
	Status         string    `json:"status"`
	Timestamp      time.Time `json:"timestamp"`
}

func NewEventGenerator(kafkaService input.KafkaServiceInterface, topic string) *EventGenerator {
	return &EventGenerator{
		kafkaService: kafkaService,
		topic:        topic,
		stopChan:     make(chan struct{}),
	}
}

func (eg *EventGenerator) Start() {
	slog.Info("starting event generator", "interval", "60m", "batch_size", 30)

	// Wait for Kafka consumer to subscribe before sending initial batch
	slog.Info("waiting for Kafka consumer to be ready", "delay", "5s")
	time.Sleep(15 * time.Second)

	eg.generateAndSendEvents()

	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			eg.generateAndSendEvents()
		case <-eg.stopChan:
			slog.Info("stopping event generator")
			return
		}
	}
}

func (eg *EventGenerator) generateAndSendEvents() {
	slog.Info("generating events batch", "count", 30)

	type PlantInfo struct {
		ID   uuid.UUID
		Name string
	}

	plants := []PlantInfo{
		{ID: uuid.MustParse("1e2d3c4b-5a6f-7e8d-9c0b-1a2b3c4d5e6f"), Name: "Solar Plant Alpha"},
		{ID: uuid.MustParse("2f3e4d5c-6b7a-8c9d-0e1f-2b3c4d5e6f7a"), Name: "Wind Farm Beta"},
		{ID: uuid.MustParse("c2e78b94-76f4-49a4-b6e8-d62c8d1d23ea"), Name: "Hydro Plant Gamma"},
	}

	statuses := []string{"operational", "maintenance", "standby", "peak_load"}
	eventTypes := []string{"power_reading", "status_update", "efficiency_report", "alert"}

	for i := 0; i < 30; i++ {
		selectedPlant := plants[rand.Intn(len(plants))]

		event := EnergyMonitoringEvent{
			PlantID:        fmt.Sprintf("plant-%d", rand.Intn(len(plants))+1),
			PlantSourceId:  selectedPlant.ID,
			PlantName:      selectedPlant.Name,
			EventType:      eventTypes[rand.Intn(len(eventTypes))],
			PowerGenerated: rand.Float64() * 1000,
			PowerConsumed:  rand.Float64() * 50,
			Efficiency:     75 + rand.Float64()*20,
			Temperature:    20 + rand.Float64()*30,
			Status:         statuses[rand.Intn(len(statuses))],
			Timestamp:      time.Now(),
		}

		key := fmt.Sprintf("%s-%d", event.PlantID, time.Now().Unix())
		err := eg.kafkaService.SendEvent(eg.topic, key, event)
		if err != nil {
			slog.Error("error sending event to Kafka",
				"event_num", i+1,
				"error", err,
			)
		} else {
			slog.Info("event sent",
				"event_num", i+1,
				"plant_id", event.PlantID,
				"type", event.EventType,
				"power_mw", fmt.Sprintf("%.2f", event.PowerGenerated),
			)
		}

		time.Sleep(100 * time.Millisecond)
	}

	slog.Info("finished sending events batch", "count", 30)
}

func (eg *EventGenerator) Stop() {
	close(eg.stopChan)
}
