-- +goose Up
-- Enable native compression on analytical.events_ts hypertable

-- 1. Configure compression settings
ALTER TABLE analytical.events_ts SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'plant_source_id',
    timescaledb.compress_orderby = 'created_at DESC'
);

-- 2. Add compression policy: compress chunks older than 7 days
SELECT add_compression_policy('analytical.events_ts', INTERVAL '7 days');

-- +goose Down
SELECT remove_compression_policy('analytical.events_ts');
ALTER TABLE analytical.events_ts SET (timescaledb.compress = false);
