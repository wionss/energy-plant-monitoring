package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"monitoring-energy-service/internal/domain/entities"
	"monitoring-energy-service/internal/domain/ports/input"
	"monitoring-energy-service/internal/domain/ports/output"
	"monitoring-energy-service/internal/infrastructure/adapters/telegram"

	"github.com/google/uuid"
)

// IntakeHandler procesa mensajes consumidos desde Kafka
//
// PROPÓSITO:
// Actúa como consumer de Kafka para el topic "intake", recibiendo eventos y
// guardándolos en PostgreSQL para análisis posterior.
//
// CAMBIO REALIZADO: Refactorizado para usar DualEventWriter (escritura dual)
// RAZÓN: Arquitectura multi-esquema con operational y analytical
type IntakeHandler struct {
	dualWriter            output.DualEventWriterInterface       // Para escritura dual (operational + analytical)
	energyPlantRepository output.EnergyPlantRepositoryInterface // Para validar que las plantas existen
	telegramNotifier      *telegram.Notifier                    // Para notificar errores de validación
	useAsyncWrite         bool                                  // true = async (mejor throughput), false = sync (garantía)
}

var _ input.MessageHandler = &IntakeHandler{}

// NewIntakeHandler crea una nueva instancia del handler de Kafka
// CAMBIO: Ahora recibe dualWriter para escritura dual
// RAZÓN: Arquitectura multi-esquema con operational y analytical
func NewIntakeHandler(
	dualWriter output.DualEventWriterInterface,
	energyPlantRepository output.EnergyPlantRepositoryInterface,
	telegramNotifier *telegram.Notifier,
	useAsyncWrite bool,
) *IntakeHandler {
	return &IntakeHandler{
		dualWriter:            dualWriter,
		energyPlantRepository: energyPlantRepository,
		telegramNotifier:      telegramNotifier,
		useAsyncWrite:         useAsyncWrite,
	}
}

// HandleMessage procesa cada mensaje recibido desde Kafka
//
// FLUJO:
// 1. Recibe mensaje como bytes desde Kafka
// 2. Deserializa el JSON a un mapa
// 3. Extrae event_type y plant_name
// 4. Convierte los datos a JSON string
// 5. Crea una entidad EventEntity
// 6. Guarda en PostgreSQL usando el repositorio
//
// CAMBIO REALIZADO: Completamente reescrito desde el TODO inicial
// RAZÓN: Implementar la persistencia de eventos en PostgreSQL
func (h *IntakeHandler) HandleMessage(message []byte) error {
	// CAMBIO: Log safe metadata instead of raw message to avoid exposing PII
	// RAZÓN: Evita exponer datos sensibles en logs, usa hash y tamaño del mensaje
	hash := sha256.Sum256(message)
	hashHex := hex.EncodeToString(hash[:])

	// Preview: primeros 50 bytes (o menos si el mensaje es más corto)
	previewLen := 50
	if len(message) < previewLen {
		previewLen = len(message)
	}
	preview := string(message[:previewLen])
	if len(message) > previewLen {
		preview += "..."
	}

	log.Printf("Received message on intake topic - Size: %d bytes, SHA256: %s, Preview: %s",
		len(message), hashHex, preview)

	// CAMBIO: Parse del mensaje JSON
	// RAZÓN: Necesitamos extraer campos específicos (event_type, plant_name)
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return err
	}

	// CAMBIO: Log solo campos seguros después del parsing
	// RAZÓN: Permite debugging sin exponer payload completo con posible PII
	safeEventType := "unknown"
	if et, ok := data["event_type"].(string); ok {
		safeEventType = et
	}
	safePlantName := "not_specified"
	if pn, ok := data["plant_name"].(string); ok {
		safePlantName = pn
	}
	safePlantSourceId := "not_specified"
	if psid, ok := data["plant_source_id"].(string); ok {
		safePlantSourceId = psid
	}
	log.Printf("Parsed safe fields - EventType: %s, PlantName: %s, PlantSourceId: %s",
		safeEventType, safePlantName, safePlantSourceId)

	// CAMBIO: Extrae event_type del mensaje
	// RAZÓN: Indexamos por event_type para filtrado rápido en queries
	eventType := "unknown"
	if et, ok := data["event_type"].(string); ok {
		eventType = et
	}

	// CAMBIO: Extrae plant_name como source
	// RAZÓN: Permite identificar de qué planta viene cada evento
	source := "kafka-intake"
	if plantName, ok := data["plant_name"].(string); ok {
		source = plantName
	}

	// CAMBIO: Extrae plant_source_id del mensaje
	// RAZÓN: Necesitamos el UUID de la planta para relacionar el evento con la tabla energy_plants
	var plantSourceId uuid.UUID
	if plantSourceIdStr, ok := data["plant_source_id"].(string); ok {
		parsedUUID, err := uuid.Parse(plantSourceIdStr)
		if err != nil {
			log.Printf("ERROR: Invalid plant_source_id format: %v - Message will be retried or sent to DLQ", err)

			// Notificar error de UUID inválido a Telegram
			if notifyErr := h.telegramNotifier.SendUUIDError(
				"plant_source_id",
				plantSourceIdStr,
				fmt.Sprintf("Error al parsear UUID: %v. EventType: %s, PlantName: %s", err, safeEventType, safePlantName),
			); notifyErr != nil {
				log.Printf("Failed to send Telegram notification: %v", notifyErr)
			}

			return fmt.Errorf("invalid plant_source_id format: %v", err)
		}
		plantSourceId = parsedUUID
	} else {
		log.Printf("ERROR: plant_source_id not found in message - Message will be retried or sent to DLQ")

		// Notificar campo faltante a Telegram
		if notifyErr := h.telegramNotifier.SendValidationError(
			"plant_source_id",
			"campo ausente",
			fmt.Sprintf("El campo plant_source_id no está presente en el mensaje. EventType: %s, PlantName: %s", safeEventType, safePlantName),
		); notifyErr != nil {
			log.Printf("Failed to send Telegram notification: %v", notifyErr)
		}

		return fmt.Errorf("missing plant_source_id field in message")
	}

	// CAMBIO: Validar que la planta existe en la base de datos
	// RAZÓN: Solo guardamos eventos de plantas válidas para mantener integridad referencial
	exists, err := h.energyPlantRepository.Exists(plantSourceId)
	if err != nil {
		log.Printf("ERROR: Failed to validate plant existence for plant_source_id=%s: %v", plantSourceId, err)
		return err
	}
	if !exists {
		log.Printf("ERROR: Event rejected - plant_source_id=%s does not exist in database. EventType=%s, Source=%s - Message will be retried or sent to DLQ",
			plantSourceId, eventType, source)

		// Notificar planta inexistente a Telegram
		if notifyErr := h.telegramNotifier.SendValidationError(
			"plant_source_id",
			plantSourceId.String(),
			fmt.Sprintf("La planta con ID %s no existe en la base de datos. EventType: %s, Source: %s", plantSourceId, eventType, source),
		); notifyErr != nil {
			log.Printf("Failed to send Telegram notification: %v", notifyErr)
		}

		return fmt.Errorf("plant_source_id=%s does not exist in database (eventType=%s, source=%s)",
			plantSourceId, eventType, source)
	}

	log.Printf("✓ Plant validated successfully: plant_source_id=%s", plantSourceId)

	// CAMBIO: Convierte data completo a JSON string
	// RAZÓN: PostgreSQL almacena el JSON completo como texto para consultas posteriores
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return err
	}

	// CAMBIO: Genera ID y timestamp una sola vez para ambas tablas
	// RAZÓN: Garantiza que el mismo evento tenga el mismo ID en operational y analytical
	now := time.Now()
	eventId := uuid.New()

	// CAMBIO: Crea entidad operacional (datos calientes)
	// RAZÓN: Para consultas frecuentes con FK a energy_plants
	eventOp := &entities.EventOperational{
		ID:            eventId,
		EventType:     eventType,
		PlantSourceId: plantSourceId,
		Source:        source,
		Data:          string(dataJSON),
		CreatedAt:     now,
	}

	// CAMBIO: Crea entidad analítica (datos fríos - TimescaleDB)
	// RAZÓN: Para análisis temporal y agregaciones con time_bucket
	eventAn := &entities.EventAnalytical{
		CreatedAt:     now,
		ID:            eventId,
		EventType:     eventType,
		PlantSourceId: plantSourceId,
		Source:        source,
		Data:          string(dataJSON),
	}

	// CAMBIO: Usa DualEventWriter para guardar en ambas tablas
	// RAZÓN: Arquitectura multi-esquema (operational + analytical)
	if h.useAsyncWrite {
		if err := h.dualWriter.SaveEventAsync(eventOp, eventAn); err != nil {
			log.Printf("Error saving event async: %v", err)
			return err
		}
		log.Printf("Event enqueued for async save: ID=%s, Type=%s", eventId, eventType)
	} else {
		if err := h.dualWriter.SaveEvent(eventOp, eventAn); err != nil {
			log.Printf("Error saving event to database: %v", err)
			return err
		}
		log.Printf("Event saved to both operational and analytical: ID=%s, Type=%s", eventId, eventType)
	}

	return nil
}
