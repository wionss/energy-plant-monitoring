package container

import (
	"log/slog"
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

type Container struct {
	db                    *gorm.DB
	cfg                   conf.Config
	KafkaService          input.KafkaServiceInterface
	WebhookAdapter        output.WebhookAdapterInterface
	ExampleRepository     output.ExampleRepositoryInterface
	EventRepository       output.EventRepositoryInterface
	EventOperationalRepo  output.EventOperationalRepositoryInterface
	EventAnalyticalRepo   output.EventAnalyticalRepositoryInterface
	DualEventWriter       output.DualEventWriterInterface
	EnergyPlantRepository output.EnergyPlantRepositoryInterface
	PlantStatusRepository output.PlantStatusRepositoryInterface
	AnalyticsCoordinator  output.AnalyticsCoordinatorInterface
	TelegramNotifier      *telegram.Notifier
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

	eventRepository := repositories.NewEventRepository(db)
	container.EventRepository = eventRepository

	energyPlantRepository := repositories.NewEnergyPlantRepository(db)
	container.EnergyPlantRepository = energyPlantRepository

	plantStatusRepository := repositories.NewPlantStatusRepository(db)
	container.PlantStatusRepository = plantStatusRepository

	eventOpRepo := repositories.NewEventOperationalRepository(db)
	container.EventOperationalRepo = eventOpRepo

	eventAnRepo := repositories.NewEventAnalyticalRepository(db)
	container.EventAnalyticalRepo = eventAnRepo

	// Initialize Kafka first (needed for spillover callback)
	kafkaFactory := kafkaconf.NewKafkaFactory(kafkaBrokers, autoOffset)
	kafkaAdapter := kafka.NewKafkaAdapter(kafkaFactory, consumerGroup)
	kafkaService := api.NewKafkaService(kafkaAdapter)
	container.KafkaService = kafkaService

	// Initialize DualEventWriter with spillover callback wired to Kafka DLQ
	dualWriter := repositories.NewDualEventWriterWithConfig(db, repositories.DualEventWriterConfig{
		OpWorkerCount: 2,
		AnWorkerCount: 2,
		SpilloverFunc: func(eventData []byte, reason string) {
			kafkaService.SendToDLQ(eventData, "spillover: "+reason)
		},
	})
	container.DualEventWriter = dualWriter

	// Initialize Webhook adapter
	webhookAdapter := webhook.NewAdapter(httpClient)
	container.WebhookAdapter = webhookAdapter

	// Initialize Analytics Coordinator
	analyticsWorkerRepo := repositories.NewAnalyticsWorkerRepo(db)
	analyticsCoordinator := repositories.NewAnalyticsCoordinator(
		analyticsWorkerRepo,
		webhookAdapter,
		repositories.AnalyticsCoordinatorConfig{
			WebhookURL:     container.cfg.WebhookUrl,
			WebhookEnabled: container.cfg.WebhookEnabled,
		},
	)
	container.AnalyticsCoordinator = analyticsCoordinator
	go analyticsCoordinator.Start()

	// Initialize Telegram notifier
	telegramNotifier := telegram.NewNotifier(
		container.cfg.TelegramBotToken,
		container.cfg.TelegramChatID,
		container.cfg.TelegramEnabled,
	)
	container.TelegramNotifier = telegramNotifier

	// Register Kafka handlers
	intakeHandler := api.NewIntakeHandler(dualWriter, energyPlantRepository, plantStatusRepository, telegramNotifier, true)
	kafkaService.RegisterHandler(container.cfg.ConsumerTopic, intakeHandler)

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

// Shutdown performs an ordered shutdown of all components.
func (c *Container) Shutdown() {
	slog.Info("shutting down application components")

	// 1. Stop Kafka consumer
	slog.Info("stopping Kafka consumer")
	c.KafkaService.StopConsuming()

	// 2. Stop analytics coordinator
	slog.Info("stopping analytics coordinator")
	c.AnalyticsCoordinator.Stop()

	// 3. Stop dual event writer (drains async channel)
	slog.Info("stopping dual event writer")
	c.DualEventWriter.Stop()

	// 4. Stop Telegram notifier (drains alert channel)
	slog.Info("stopping Telegram notifier")
	c.TelegramNotifier.Stop()

	// 5. Close database connection
	slog.Info("closing database connection")
	sqlDB, err := c.db.DB()
	if err != nil {
		slog.Error("failed to get underlying sql.DB", "error", err)
	} else {
		if err := sqlDB.Close(); err != nil {
			slog.Error("failed to close database", "error", err)
		}
	}

	slog.Info("shutdown complete")
}
