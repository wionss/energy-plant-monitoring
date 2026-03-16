package repositories

import (
	"fmt"
	"math/rand"
	"time"

	"monitoring-energy-service/internal/domain/entities"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AnalyticsWorkerRepo implements the analytics aggregation repository
type AnalyticsWorkerRepo struct {
	db *gorm.DB
}

var _ output.AnalyticsWorkerRepoInterface = &AnalyticsWorkerRepo{}

// NewAnalyticsWorkerRepo creates a new analytics worker repository
func NewAnalyticsWorkerRepo(db *gorm.DB) *AnalyticsWorkerRepo {
	return &AnalyticsWorkerRepo{db: db}
}

// RecalculateDirtyBuckets finds buckets with new data and recalculates hourly stats.
// Uses a single CTE query to:
// 1. Find dirty buckets (buckets with data in the lookback window)
// 2. Calculate stats for those buckets
// 3. Upsert into hourly_plant_stats
// 4. Queue webhooks for notification
// Returns the number of buckets recalculated.
func (r *AnalyticsWorkerRepo) RecalculateDirtyBuckets(lookbackWindow time.Duration) (int, error) {
	// Convert duration to PostgreSQL interval format
	intervalStr := fmt.Sprintf("%d minutes", int(lookbackWindow.Minutes()))

	query := `
WITH dirty_buckets AS (
    SELECT DISTINCT
        time_bucket('1 hour', created_at) AS bucket,
        plant_source_id
    FROM analytical.events_ts
    WHERE created_at >= NOW() - $1::interval
),
calculated_stats AS (
    SELECT
        time_bucket('1 hour', e.created_at) AS bucket,
        e.plant_source_id,
        -- Datos desnormalizados de master.energy_plants
        ep.plant_name,
        ep.plant_type,
        ep.latitude,
        ep.longitude,
        -- Métricas calculadas
        AVG((e.data::jsonb->>'power_generated_mw')::double precision) AS avg_power_gen,
        AVG((e.data::jsonb->>'power_consumed_mw')::double precision) AS avg_power_con,
        AVG((e.data::jsonb->>'efficiency_percent')::double precision) AS avg_efficiency,
        AVG((e.data::jsonb->>'temperature_celsius')::double precision) AS avg_temp,
        COUNT(*) AS sample_count
    FROM analytical.events_ts e
    INNER JOIN dirty_buckets db
        ON time_bucket('1 hour', e.created_at) = db.bucket
        AND e.plant_source_id = db.plant_source_id
    LEFT JOIN master.energy_plants ep
        ON e.plant_source_id = ep.id
    GROUP BY 1, 2, ep.plant_name, ep.plant_type, ep.latitude, ep.longitude
),
upserted AS (
    INSERT INTO analytical.hourly_plant_stats
        (bucket, plant_source_id, plant_name, plant_type, latitude, longitude, avg_power_gen, avg_power_con, avg_efficiency, avg_temp, sample_count, last_calculated_at)
    SELECT bucket, plant_source_id, plant_name, plant_type, latitude, longitude, avg_power_gen, avg_power_con, avg_efficiency, avg_temp, sample_count, NOW()
    FROM calculated_stats
    ON CONFLICT (bucket, plant_source_id) DO UPDATE SET
        plant_name = EXCLUDED.plant_name,
        plant_type = EXCLUDED.plant_type,
        latitude = EXCLUDED.latitude,
        longitude = EXCLUDED.longitude,
        avg_power_gen = EXCLUDED.avg_power_gen,
        avg_power_con = EXCLUDED.avg_power_con,
        avg_efficiency = EXCLUDED.avg_efficiency,
        avg_temp = EXCLUDED.avg_temp,
        sample_count = EXCLUDED.sample_count,
        last_calculated_at = NOW()
    RETURNING bucket, plant_source_id
)
INSERT INTO operational.webhook_queue (bucket, plant_source_id, status)
SELECT bucket, plant_source_id, 'PENDING' FROM upserted
ON CONFLICT (bucket, plant_source_id) DO UPDATE SET
    status = 'PENDING', attempts = 0, error_message = NULL
RETURNING id;
`

	var ids []uuid.UUID
	if err := r.db.Raw(query, intervalStr).Scan(&ids).Error; err != nil {
		return 0, fmt.Errorf("failed to recalculate dirty buckets: %w", err)
	}

	return len(ids), nil
}

// GetPendingWebhooks retrieves webhooks ready to be dispatched
func (r *AnalyticsWorkerRepo) GetPendingWebhooks(limit int) ([]*entities.WebhookQueueItem, error) {
	var models []WebhookQueueModel

	err := r.db.Where("status IN ?", []string{"PENDING", "RETRYING"}).
		Where("next_retry_at IS NULL OR next_retry_at <= NOW()").
		Order("created_at ASC").
		Limit(limit).
		Find(&models).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get pending webhooks: %w", err)
	}

	result := make([]*entities.WebhookQueueItem, len(models))
	for i := range models {
		result[i] = ToWebhookQueueEntity(&models[i])
	}

	return result, nil
}

// UpdateWebhookStatus updates a webhook's status with exponential backoff for retries
func (r *AnalyticsWorkerRepo) UpdateWebhookStatus(id uuid.UUID, status entities.WebhookStatus, errorMsg string) error {
	now := time.Now()

	updates := map[string]interface{}{
		"status":       string(status),
		"last_attempt": now,
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
	} else {
		updates["error_message"] = nil
	}

	// Calculate next retry with exponential backoff if retrying
	if status == entities.WebhookStatusRetrying {
		var model WebhookQueueModel
		if err := r.db.First(&model, "id = ?", id).Error; err != nil {
			return fmt.Errorf("failed to find webhook for retry: %w", err)
		}

		// Exponential backoff with ±33% jitter: 30s, 1min, 2min, 4min, 8min (capped)
		base := 30 * (1 << model.Attempts)
		if base > 480 {
			base = 480
		}
		jitter := int(rand.Int63n(int64(base/3) + 1))
		backoffSeconds := base + jitter
		nextRetry := now.Add(time.Duration(backoffSeconds) * time.Second)
		updates["next_retry_at"] = nextRetry
		updates["attempts"] = gorm.Expr("attempts + 1")
	} else if status == entities.WebhookStatusSent {
		updates["next_retry_at"] = nil
	}

	if err := r.db.Model(&WebhookQueueModel{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update webhook status: %w", err)
	}

	return nil
}

// GetHourlyStats retrieves the hourly stats for a specific bucket and plant
func (r *AnalyticsWorkerRepo) GetHourlyStats(bucket time.Time, plantId uuid.UUID) (*entities.HourlyPlantStats, error) {
	var model HourlyPlantStatsModel

	err := r.db.Where("bucket = ? AND plant_source_id = ?", bucket, plantId).First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get hourly stats: %w", err)
	}

	return ToHourlyPlantStatsEntity(&model), nil
}
