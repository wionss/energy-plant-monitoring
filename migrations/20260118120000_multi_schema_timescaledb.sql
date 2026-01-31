-- +goose Up
-- Migration: Multi-Schema Architecture with TimescaleDB
-- Schemas: master (datos maestros), operational (datos calientes), analytical (datos fríos)
-- NOTA: Los esquemas ya fueron creados en 20260110171050_bootstrap_schemas.sql
-- NOTA: master.energy_plants ya fue creada en la migración inicial

-- =============================================================================
-- PASO 1: Crear tabla operational.events_std (PostgreSQL estándar)
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
-- PASO 2: Crear tabla analytical.events_ts
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
-- PASO 2b: Convertir a TimescaleDB hypertable (solo si la extensión YA está instalada)
-- =============================================================================
-- NOTA: No intentamos CREATE EXTENSION aquí porque puede causar desconexión si
-- shared_preload_libraries no está configurado. TimescaleDB debe ser configurado
-- manualmente en el servidor antes de ejecutar esta migración.
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
-- PASO 3: Migrar datos existentes (si la tabla events existe)
-- =============================================================================
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'events') THEN
        -- Migrar últimos 30 días a operational
        INSERT INTO operational.events_std (id, event_type, plant_source_id, source, data, metadata, created_at)
        SELECT id, event_type, plant_source_id, source, data, metadata, created_at
        FROM public.events
        WHERE created_at >= NOW() - INTERVAL '30 days'
        ON CONFLICT (id) DO NOTHING;

        -- Migrar todo a analytical
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
-- PASO 4: Eliminar tabla legacy (opcional - comentado por seguridad)
-- =============================================================================
-- DROP TABLE IF EXISTS public.events;

-- =============================================================================
-- PASO 5: Configurar compresión para hypertable (opcional)
-- =============================================================================
-- ALTER TABLE analytical.events_ts SET (
--     timescaledb.compress,
--     timescaledb.compress_segmentby = 'plant_source_id'
-- );
-- SELECT add_compression_policy('analytical.events_ts', INTERVAL '7 days');

-- +goose Down
-- Revertir migración

-- Eliminar tablas de nuevos esquemas
DROP TABLE IF EXISTS analytical.events_ts;
DROP TABLE IF EXISTS operational.events_std;
