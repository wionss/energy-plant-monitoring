package repositories

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"monitoring-energy-service/internal/domain/entities"
	"monitoring-energy-service/internal/domain/ports/output"
)

// AnalyticsCoordinatorConfig holds configuration for the analytics coordinator
type AnalyticsCoordinatorConfig struct {
	// AggregatorInterval is how often to check for dirty buckets (default: 5 minutes)
	AggregatorInterval time.Duration
	// DispatcherInterval is how often to process the webhook queue (default: 30 seconds)
	DispatcherInterval time.Duration
	// LookbackWindow is how far back to look for dirty buckets (default: 10 minutes)
	LookbackWindow time.Duration
	// MaxWebhookAttempts is the maximum number of retry attempts (default: 5)
	MaxWebhookAttempts int
	// WebhookBatchSize is how many webhooks to process per cycle (default: 100)
	WebhookBatchSize int
	// WebhookURL is the target URL for webhook notifications
	WebhookURL string
	// WebhookEnabled controls whether webhooks are actually sent
	WebhookEnabled bool
}

// DefaultAnalyticsCoordinatorConfig returns a config with sensible defaults
func DefaultAnalyticsCoordinatorConfig() AnalyticsCoordinatorConfig {
	return AnalyticsCoordinatorConfig{
		AggregatorInterval: 5 * time.Minute,
		DispatcherInterval: 30 * time.Second,
		LookbackWindow:     10 * time.Minute,
		MaxWebhookAttempts: 5,
		WebhookBatchSize:   100,
	}
}

// AnalyticsCoordinator orchestrates the analytics aggregation and webhook dispatch
type AnalyticsCoordinator struct {
	repo           output.AnalyticsWorkerRepoInterface
	webhookAdapter output.WebhookAdapterInterface
	config         AnalyticsCoordinatorConfig

	wg       sync.WaitGroup
	stopOnce sync.Once
	stopChan chan struct{}

	// Metrics
	bucketsProcessed   atomic.Int64
	webhooksSent       atomic.Int64
	webhooksFailed     atomic.Int64
	metricsStartTime   time.Time
}

var _ output.AnalyticsCoordinatorInterface = &AnalyticsCoordinator{}

// NewAnalyticsCoordinator creates a new analytics coordinator
func NewAnalyticsCoordinator(
	repo output.AnalyticsWorkerRepoInterface,
	webhookAdapter output.WebhookAdapterInterface,
	config AnalyticsCoordinatorConfig,
) *AnalyticsCoordinator {
	// Apply defaults for zero values
	if config.AggregatorInterval == 0 {
		config.AggregatorInterval = 5 * time.Minute
	}
	if config.DispatcherInterval == 0 {
		config.DispatcherInterval = 30 * time.Second
	}
	if config.LookbackWindow == 0 {
		config.LookbackWindow = 10 * time.Minute
	}
	if config.MaxWebhookAttempts == 0 {
		config.MaxWebhookAttempts = 5
	}
	if config.WebhookBatchSize == 0 {
		config.WebhookBatchSize = 100
	}

	return &AnalyticsCoordinator{
		repo:             repo,
		webhookAdapter:   webhookAdapter,
		config:           config,
		stopChan:         make(chan struct{}),
		metricsStartTime: time.Now(),
	}
}

// Start begins the coordinator's background goroutines
func (c *AnalyticsCoordinator) Start() {
	slog.Info("starting AnalyticsCoordinator",
		"aggregator_interval", c.config.AggregatorInterval,
		"dispatcher_interval", c.config.DispatcherInterval,
		"lookback_window", c.config.LookbackWindow,
		"webhook_enabled", c.config.WebhookEnabled,
	)

	// Start aggregator
	c.wg.Add(1)
	go c.runAggregator()

	// Start dispatcher
	c.wg.Add(1)
	go c.runDispatcher()

	// Start metrics reporter
	c.wg.Add(1)
	go c.runMetricsReporter()
}

// Stop gracefully shuts down the coordinator
func (c *AnalyticsCoordinator) Stop() {
	c.stopOnce.Do(func() {
		slog.Info("stopping AnalyticsCoordinator")
		close(c.stopChan)
		c.wg.Wait()

		// Final metrics
		elapsed := time.Since(c.metricsStartTime)
		slog.Info("AnalyticsCoordinator stopped",
			"total_buckets_processed", c.bucketsProcessed.Load(),
			"total_webhooks_sent", c.webhooksSent.Load(),
			"total_webhooks_failed", c.webhooksFailed.Load(),
			"uptime_seconds", elapsed.Seconds(),
		)
	})
}

// runAggregator periodically recalculates dirty buckets
func (c *AnalyticsCoordinator) runAggregator() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.AggregatorInterval)
	defer ticker.Stop()

	// Run once immediately on start
	c.doAggregation()

	for {
		select {
		case <-c.stopChan:
			slog.Info("aggregator stopped")
			return
		case <-ticker.C:
			c.doAggregation()
		}
	}
}

func (c *AnalyticsCoordinator) doAggregation() {
	start := time.Now()

	count, err := c.repo.RecalculateDirtyBuckets(c.config.LookbackWindow)
	if err != nil {
		slog.Error("failed to recalculate dirty buckets", "error", err)
		return
	}

	if count > 0 {
		c.bucketsProcessed.Add(int64(count))
		slog.Info("dirty buckets recalculated",
			"count", count,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
}

// runDispatcher periodically processes the webhook queue
func (c *AnalyticsCoordinator) runDispatcher() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.DispatcherInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			slog.Info("dispatcher stopped")
			return
		case <-ticker.C:
			c.doDispatch()
		}
	}
}

func (c *AnalyticsCoordinator) doDispatch() {
	if !c.config.WebhookEnabled {
		return
	}

	webhooks, err := c.repo.GetPendingWebhooks(c.config.WebhookBatchSize)
	if err != nil {
		slog.Error("failed to get pending webhooks", "error", err)
		return
	}

	for _, wh := range webhooks {
		// Check if max attempts exceeded
		if wh.Attempts >= c.config.MaxWebhookAttempts {
			if err := c.repo.UpdateWebhookStatus(wh.ID, entities.WebhookStatusFailed, "max attempts exceeded"); err != nil {
				slog.Error("failed to mark webhook as failed", "id", wh.ID, "error", err)
			}
			c.webhooksFailed.Add(1)
			continue
		}

		// Get the stats for the webhook payload
		stats, err := c.repo.GetHourlyStats(wh.Bucket, wh.PlantSourceId)
		if err != nil {
			slog.Error("failed to get hourly stats for webhook", "id", wh.ID, "error", err)
			if err := c.repo.UpdateWebhookStatus(wh.ID, entities.WebhookStatusRetrying, err.Error()); err != nil {
				slog.Error("failed to update webhook status", "id", wh.ID, "error", err)
			}
			continue
		}

		if stats == nil {
			// Stats not found, mark as failed
			if err := c.repo.UpdateWebhookStatus(wh.ID, entities.WebhookStatusFailed, "stats not found"); err != nil {
				slog.Error("failed to mark webhook as failed", "id", wh.ID, "error", err)
			}
			c.webhooksFailed.Add(1)
			continue
		}

		// Build payload
		payload := map[string]any{
			"bucket":           stats.Bucket,
			"plant_source_id":  stats.PlantSourceId,
			"avg_power_gen":    stats.AvgPowerGen,
			"avg_power_con":    stats.AvgPowerCon,
			"avg_efficiency":   stats.AvgEfficiency,
			"avg_temp":         stats.AvgTemp,
			"sample_count":     stats.SampleCount,
			"calculated_at":    stats.LastCalculatedAt,
		}

		// Send webhook
		if err := c.webhookAdapter.SendPayload(c.config.WebhookURL, payload); err != nil {
			slog.Warn("webhook send failed",
				"id", wh.ID,
				"bucket", wh.Bucket,
				"plant_id", wh.PlantSourceId,
				"attempt", wh.Attempts+1,
				"error", err,
			)
			if err := c.repo.UpdateWebhookStatus(wh.ID, entities.WebhookStatusRetrying, err.Error()); err != nil {
				slog.Error("failed to update webhook status", "id", wh.ID, "error", err)
			}
			continue
		}

		// Success
		if err := c.repo.UpdateWebhookStatus(wh.ID, entities.WebhookStatusSent, ""); err != nil {
			slog.Error("failed to mark webhook as sent", "id", wh.ID, "error", err)
		}
		c.webhooksSent.Add(1)

		slog.Debug("webhook sent successfully",
			"id", wh.ID,
			"bucket", wh.Bucket,
			"plant_id", wh.PlantSourceId,
		)
	}
}

// runMetricsReporter periodically logs metrics
func (c *AnalyticsCoordinator) runMetricsReporter() {
	defer c.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	var lastBuckets, lastSent, lastFailed int64

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			buckets := c.bucketsProcessed.Load()
			sent := c.webhooksSent.Load()
			failed := c.webhooksFailed.Load()

			slog.Info("analytics_coordinator_metrics",
				"buckets_total", buckets,
				"buckets_last_60s", buckets-lastBuckets,
				"webhooks_sent_total", sent,
				"webhooks_sent_last_60s", sent-lastSent,
				"webhooks_failed_total", failed,
				"webhooks_failed_last_60s", failed-lastFailed,
			)

			lastBuckets = buckets
			lastSent = sent
			lastFailed = failed
		}
	}
}
