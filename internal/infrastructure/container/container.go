package container

import (
	"net/http"

	"monitoring-energy-service/internal/api"
	"monitoring-energy-service/internal/domain/ports/input"
	"monitoring-energy-service/internal/domain/ports/output"
	"monitoring-energy-service/internal/infrastructure/adapters/http/webhook"
	"monitoring-energy-service/internal/infrastructure/adapters/kafka"
	"monitoring-energy-service/internal/infrastructure/adapters/repositories"
	"monitoring-energy-service/internal/infrastructure/adapters/telegram"
	"monitoring-energy-service/internal/infrastructure/conf"
	"monitoring-energy-service/internal/infrastructure/conf/kafkaconf"

	"gorm.io/gorm"
)

type ContainerOption func(*Container)

// Container mantiene todas las dependencias de la aplicación (Dependency Injection)
//
// CAMBIOS REALIZADOS:
// - Agregado EventRepository: Para acceso a base de datos de eventos (legacy)
// - Agregado EventOperationalRepo: Para datos calientes (operational.events_std)
// - Agregado EventAnalyticalRepo: Para datos fríos (analytical.events_ts)
// - Agregado DualEventWriter: Para escritura dual a ambas tablas
// - Agregado EnergyPlantRepository: Para validar plantas antes de guardar eventos
// - Agregado TelegramNotifier: Para notificar errores de validación a Telegram
type Container struct {
	db                    *gorm.DB
	cfg                   conf.Config
	KafkaService          input.KafkaServiceInterface
	WebhookAdapter        output.WebhookAdapterInterface
	ExampleRepository     output.ExampleRepositoryInterface
	EventRepository       output.EventRepositoryInterface           // Legacy - Para gestionar eventos en DB
	EventOperationalRepo  output.EventOperationalRepositoryInterface // Datos calientes
	EventAnalyticalRepo   output.EventAnalyticalRepositoryInterface  // Datos fríos (TimescaleDB)
	DualEventWriter       output.DualEventWriterInterface            // Escritura dual
	EnergyPlantRepository output.EnergyPlantRepositoryInterface      // Para validar plantas
	EventGenerator        *api.EventGenerator                        // Para generar eventos cada 5 min
	TelegramNotifier      *telegram.Notifier                         // Para notificar errores a Telegram
}

func NewContainer(
	db *gorm.DB,
	kafkaBrokers []string,
	consumerGroup string,
	httpClient *http.Client,
	autoOffset string,
	opts ...ContainerOption,
) *Container {
	container := &Container{db: db}

	for _, opt := range opts {
		opt(container)
	}

	// Initialize repositories
	exampleRepository := repositories.NewExampleRepository(db)
	container.ExampleRepository = exampleRepository

	// CAMBIO: Inicializa EventRepository (legacy)
	// RAZÓN: Mantiene compatibilidad con API REST existente
	eventRepository := repositories.NewEventRepository(db)
	container.EventRepository = eventRepository

	// CAMBIO: Inicializa EnergyPlantRepository
	// RAZÓN: Necesario para validar que las plantas existen antes de guardar eventos
	energyPlantRepository := repositories.NewEnergyPlantRepository(db)
	container.EnergyPlantRepository = energyPlantRepository

	// CAMBIO: Inicializa repositorios multi-esquema
	// RAZÓN: Arquitectura con operational (datos calientes) y analytical (datos fríos)
	eventOpRepo := repositories.NewEventOperationalRepository(db)
	container.EventOperationalRepo = eventOpRepo

	eventAnRepo := repositories.NewEventAnalyticalRepository(db)
	container.EventAnalyticalRepo = eventAnRepo

	// CAMBIO: Inicializa DualEventWriter con 4 workers para async
	// RAZÓN: Permite escritura dual a operational y analytical
	dualWriter := repositories.NewDualEventWriter(db, eventOpRepo, eventAnRepo, 4)
	container.DualEventWriter = dualWriter

	// Initialize Kafka
	kafkaFactory := kafkaconf.NewKafkaFactory(kafkaBrokers, autoOffset)
	kafkaAdapter := kafka.NewKafkaAdapter(kafkaFactory, consumerGroup)
	kafkaService := api.NewKafkaService(kafkaAdapter)
	container.KafkaService = kafkaService

	// Initialize Webhook adapter
	webhookAdapter := webhook.NewAdapter(httpClient)
	container.WebhookAdapter = webhookAdapter

	// Initialize Telegram notifier
	telegramNotifier := telegram.NewNotifier(
		container.cfg.TelegramBotToken,
		container.cfg.TelegramChatID,
		container.cfg.TelegramEnabled,
	)
	container.TelegramNotifier = telegramNotifier

	// Register Kafka handlers here
	// CAMBIO: IntakeHandler ahora usa DualEventWriter para escritura dual
	// RAZÓN: Arquitectura multi-esquema (operational + analytical)
	intakeHandler := api.NewIntakeHandler(dualWriter, energyPlantRepository, telegramNotifier, true)
	kafkaService.RegisterHandler(container.cfg.ConsumerTopic, intakeHandler)

	// CAMBIO: Inicializa Event Generator con topic "intake"
	// RAZÓN: Genera automáticamente 30 eventos cada 5 minutos enviándolos a Kafka
	eventGenerator := api.NewEventGenerator(kafkaService, "intake")
	container.EventGenerator = eventGenerator

	return container
}

func WithConfig(config conf.Config) ContainerOption {
	return func(c *Container) {
		c.cfg = config
	}
}

func (c *Container) GetConfig() conf.Config {
	return c.cfg
}
