-- +goose NO TRANSACTION
-- +goose Up
-- Paso 6: TimescaleDB Continuous Aggregates for Analytics Performance
-- Pre-aggregates hourly event statistics to avoid repeated calculations
-- NOTE: named events_hourly_agg to avoid collision with the worker's hourly_plant_stats table

-- Drop if exists to ensure clean state on retry.
-- TimescaleDB with materialized_only=false stores the CA as a regular VIEW in pg_views,
-- so DROP MATERIALIZED VIEW may fail. We fall back to dropping the internal materialized
-- hypertable directly, which cascades through TimescaleDB's catalog via foreign keys.
-- +goose StatementBegin
DO $$
DECLARE
    v_mat_schema text;
    v_mat_table  text;
BEGIN
    -- Attempt 1: let TimescaleDB's hook do a clean drop (works in most 2.x versions)
    BEGIN
        EXECUTE 'DROP MATERIALIZED VIEW IF EXISTS analytical.events_hourly_agg CASCADE';
    EXCEPTION WHEN OTHERS THEN
        NULL; -- ignore: object may be stored as a regular VIEW
    END;

    -- Attempt 2: if the CA is still registered in TimescaleDB's catalog,
    -- drop the internal materialized hypertable — FK cascades clean the catalog
    SELECT h.schema_name, h.table_name
    INTO   v_mat_schema, v_mat_table
    FROM   _timescaledb_catalog.continuous_agg ca
    JOIN   _timescaledb_catalog.hypertable h ON h.id = ca.mat_hypertable_id
    WHERE  ca.user_view_schema = 'analytical'
    AND    ca.user_view_name   = 'events_hourly_agg';

    IF FOUND THEN
        EXECUTE format('DROP TABLE IF EXISTS %I.%I CASCADE', v_mat_schema, v_mat_table);
    END IF;

    -- Attempt 3: clean up the user-facing object if it still exists
    IF EXISTS (SELECT 1 FROM pg_views    WHERE schemaname = 'analytical' AND viewname    = 'events_hourly_agg') THEN
        EXECUTE 'DROP VIEW analytical.events_hourly_agg CASCADE';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_matviews WHERE schemaname = 'analytical' AND matviewname = 'events_hourly_agg') THEN
        EXECUTE 'DROP MATERIALIZED VIEW analytical.events_hourly_agg CASCADE';
    END IF;
END;
$$;
-- +goose StatementEnd

-- Create continuous aggregate for hourly plant event statistics
-- +goose StatementBegin
CREATE MATERIALIZED VIEW analytical.events_hourly_agg
WITH (timescaledb.continuous, timescaledb.materialized_only=false)
AS
SELECT
    time_bucket('1 hour', created_at) as bucket_hour,
    plant_source_id,
    event_type,
    count(*) as event_count,
    count(DISTINCT source) as distinct_sources,
    count(DISTINCT date_trunc('minute', created_at)) as distinct_minutes
FROM analytical.events_ts
GROUP BY time_bucket('1 hour', created_at), plant_source_id, event_type
WITH DATA;
-- +goose StatementEnd

-- Create index for efficient queries by plant and time range
CREATE INDEX idx_hourly_stats_plant_time
ON analytical.events_hourly_agg (plant_source_id, bucket_hour DESC);

-- Create index for filtering by event type
CREATE INDEX idx_hourly_stats_event_type
ON analytical.events_hourly_agg (event_type, bucket_hour DESC);

-- Configure automatic refresh policy (refresh every hour, covers last 3 days)
SELECT add_continuous_aggregate_policy('analytical.events_hourly_agg',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour'
) AS policy_id;

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    v_mat_schema text;
    v_mat_table  text;
BEGIN
    BEGIN
        EXECUTE 'DROP MATERIALIZED VIEW IF EXISTS analytical.events_hourly_agg CASCADE';
    EXCEPTION WHEN OTHERS THEN
        NULL;
    END;

    SELECT h.schema_name, h.table_name
    INTO   v_mat_schema, v_mat_table
    FROM   _timescaledb_catalog.continuous_agg ca
    JOIN   _timescaledb_catalog.hypertable h ON h.id = ca.mat_hypertable_id
    WHERE  ca.user_view_schema = 'analytical'
    AND    ca.user_view_name   = 'events_hourly_agg';

    IF FOUND THEN
        EXECUTE format('DROP TABLE IF EXISTS %I.%I CASCADE', v_mat_schema, v_mat_table);
    END IF;

    IF EXISTS (SELECT 1 FROM pg_views    WHERE schemaname = 'analytical' AND viewname    = 'events_hourly_agg') THEN
        EXECUTE 'DROP VIEW analytical.events_hourly_agg CASCADE';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_matviews WHERE schemaname = 'analytical' AND matviewname = 'events_hourly_agg') THEN
        EXECUTE 'DROP MATERIALIZED VIEW analytical.events_hourly_agg CASCADE';
    END IF;
END;
$$;
-- +goose StatementEnd
