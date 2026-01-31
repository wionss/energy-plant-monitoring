-- +goose Up
-- Migration: Analytics Worker tables for hourly aggregation and webhook dispatch

-- =============================================================================
-- hourly_plant_stats: Pre-calculated hourly statistics per plant
-- =============================================================================
CREATE TABLE IF NOT EXISTS analytical.hourly_plant_stats (
    bucket timestamptz NOT NULL,
    plant_source_id uuid NOT NULL,
    avg_power_gen double precision,
    avg_power_con double precision,
    avg_efficiency double precision,
    avg_temp double precision,
    sample_count bigint NOT NULL DEFAULT 0,
    last_calculated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (bucket, plant_source_id)
);

-- =============================================================================
-- webhook_queue: Dispatch queue for webhook notifications
-- =============================================================================
CREATE TABLE IF NOT EXISTS operational.webhook_queue (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket timestamptz NOT NULL,
    plant_source_id uuid NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'PENDING',
    attempts bigint NOT NULL DEFAULT 0,
    last_attempt timestamptz,
    next_retry_at timestamptz,
    error_message text,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Indexes for webhook_queue
CREATE INDEX IF NOT EXISTS idx_wq_status ON operational.webhook_queue(status);
CREATE INDEX IF NOT EXISTS idx_wq_next_retry ON operational.webhook_queue(next_retry_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wq_bucket_plant ON operational.webhook_queue(bucket, plant_source_id);

-- +goose Down
DROP TABLE IF EXISTS operational.webhook_queue;
DROP TABLE IF EXISTS analytical.hourly_plant_stats;
