-- +goose Up
-- Migration: Multi-Schema Architecture with TimescaleDB
-- Schemas: master (master data), operational (hot data), analytical (cold data)
-- NOTE: Schemas were already created in 20260110171050_bootstrap_schemas.sql
-- NOTE: master.energy_plants was already created in the initial migration

-- =============================================================================
-- STEP 1: Create operational.events_std table (standard PostgreSQL)
-- =============================================================================
CREATE TABLE IF NOT EXISTS operational.events_std (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type varchar(100) NOT NULL,
    plant_source_id uuid NOT NULL REFERENCES master.energy_plants(id),
    source varchar(255),
    data jsonb NOT NULL,
    metadata jsonb,
    created_at timestamptz DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_op_event_type ON operational.events_std(event_type);
CREATE INDEX IF NOT EXISTS idx_op_plant_source_id ON operational.events_std(plant_source_id);
CREATE INDEX IF NOT EXISTS idx_op_created_at ON operational.events_std(created_at DESC);

-- =============================================================================
-- STEP 2: Create analytical.events_ts table
-- =============================================================================
CREATE TABLE IF NOT EXISTS analytical.events_ts (
    created_at timestamptz NOT NULL DEFAULT now(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    event_type varchar(100) NOT NULL,
    plant_source_id uuid NOT NULL,
    source varchar(255),
    data jsonb NOT NULL,
    metadata jsonb,
    PRIMARY KEY (created_at, id)
);

CREATE INDEX IF NOT EXISTS idx_an_event_type ON analytical.events_ts(event_type);
CREATE INDEX IF NOT EXISTS idx_an_plant_source_id ON analytical.events_ts(plant_source_id);

-- =============================================================================
-- STEP 2b: Convert to TimescaleDB hypertable (only if the extension is already installed)
-- =============================================================================
-- NOTE: We do not attempt CREATE EXTENSION here because it may cause a disconnection if
-- shared_preload_libraries is not configured. TimescaleDB must be configured on the server
-- before running this migration.
-- +goose StatementBegin
DO $$
BEGIN
    -- Only use TimescaleDB if it's ALREADY installed (not just available)
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('analytical.events_ts', 'created_at',
            chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);
        RAISE NOTICE 'Hypertable created for analytical.events_ts';
    ELSE
        RAISE NOTICE 'TimescaleDB not installed, analytical.events_ts remains a regular table';
    END IF;
END $$;
-- +goose StatementEnd

-- =============================================================================
-- STEP 3: Migrate existing data (if the table events exists)
-- =============================================================================
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'events') THEN
        -- Migrate last 30 days of data to operational.events_std
        INSERT INTO operational.events_std (id, event_type, plant_source_id, source, data, metadata, created_at)
        SELECT id, event_type, plant_source_id, source, data, metadata, created_at
        FROM public.events
        WHERE created_at >= NOW() - INTERVAL '30 days'
        ON CONFLICT (id) DO NOTHING;

        -- Migrate all data to analytical.events_ts (if hypertable, it will handle chunking)
        INSERT INTO analytical.events_ts (created_at, id, event_type, plant_source_id, source, data, metadata)
        SELECT created_at, id, event_type, plant_source_id, source, data, metadata
        FROM public.events
        ON CONFLICT DO NOTHING;

        RAISE NOTICE 'Datos migrados desde public.events';
    ELSE
        RAISE NOTICE 'Tabla public.events no existe, saltando migración de datos';
    END IF;
END $$;
-- +goose StatementEnd

-- =============================================================================
-- STEP 4: Drop legacy table (optional - commented out for safety)
-- =============================================================================
-- DROP TABLE IF EXISTS public.events;

-- =============================================================================
-- STEP 5: Configure compression for hypertable (optional)
-- =============================================================================
-- ALTER TABLE analytical.events_ts SET (
--     timescaledb.compress,
--     timescaledb.compress_segmentby = 'plant_source_id'
-- );
-- SELECT add_compression_policy('analytical.events_ts', INTERVAL '7 days');

-- +goose Down
-- Revert migration: Drop new tables and indexes, but keep master.energy_plants intact

-- Eliminate analytical and operational tables
DROP TABLE IF EXISTS analytical.events_ts;
DROP TABLE IF EXISTS operational.events_std;
