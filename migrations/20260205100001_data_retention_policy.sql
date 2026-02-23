-- +goose Up
-- Paso 8: Data Retention Policy
-- Prevents unlimited disk growth by automatically removing old event data

-- Add 30-day retention policy to analytical events hypertable
-- This is safe because hourly_plant_stats captures aggregations before deletion
SELECT add_retention_policy(
    'analytical.events_ts',
    INTERVAL '30 days',
    if_not_exists => true
) AS policy_id;

-- Add 90-day retention policy to operational events (more data)
SELECT add_retention_policy(
    'operational.events_std',
    INTERVAL '90 days',
    if_not_exists => true
) AS policy_id;

-- +goose Down
-- Remove retention policies if this migration is rolled back
SELECT remove_retention_policy(
    'analytical.events_ts',
    if_exists => true
);

SELECT remove_retention_policy(
    'operational.events_std',
    if_exists => true
);
