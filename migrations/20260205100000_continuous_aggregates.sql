-- +goose Up
-- Paso 6: TimescaleDB Continuous Aggregates for Analytics Performance
-- Pre-aggregates hourly event statistics to avoid repeated calculations

-- Create continuous aggregate for hourly plant event statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS analytical.hourly_plant_stats
WITH (timescaledb.continuous, timescaledb.materialized_only=false)
AS
SELECT
    time_bucket('1 hour', created_at) as hour,
    plant_source_id,
    event_type,
    count(*) as event_count,
    count(DISTINCT source) as distinct_sources,
    count(DISTINCT date_trunc('minute', created_at)) as distinct_minutes
FROM analytical.events_ts
GROUP BY hour, plant_source_id, event_type
WITH DATA;

-- Create index for efficient queries by plant and time range
CREATE INDEX idx_hourly_stats_plant_time
ON analytical.hourly_plant_stats (plant_source_id, hour DESC);

-- Create index for filtering by event type
CREATE INDEX idx_hourly_stats_event_type
ON analytical.hourly_plant_stats (event_type, hour DESC);

-- Configure automatic refresh policy (refresh every hour, with 30 minutes lag)
SELECT add_continuous_aggregate_policy(
    'analytical.hourly_plant_stats',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '30 minutes',
    schedule_interval => INTERVAL '1 hour'
) AS policy_id;

-- +goose Down
-- Drop continuous aggregate with cascading policies
SELECT remove_continuous_aggregate_policy('analytical.hourly_plant_stats', if_exists => true);
DROP MATERIALIZED VIEW IF EXISTS analytical.hourly_plant_stats CASCADE;
