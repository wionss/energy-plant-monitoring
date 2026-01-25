-- +goose Up
-- Migration: Multi-Schema Architecture with TimescaleDB
-- Schemas: master (datos maestros), operational (datos calientes), analytical (datos fríos)
-- NOTA: Los esquemas ya fueron creados en 20260110171050_bootstrap_schemas.sql

-- =============================================================================
-- PASO 1: Mover energy_plants a esquema master (si existe en public)
-- =============================================================================
-- +goose StatementBegin
DO $$
BEGIN
    -- Only move if source exists AND destination doesn't exist
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'energy_plants')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'master' AND table_name = 'energy_plants') THEN
        ALTER TABLE public.energy_plants SET SCHEMA master;
        RAISE NOTICE 'Tabla energy_plants movida a esquema master';
    ELSIF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'master' AND table_name = 'energy_plants') THEN
        RAISE NOTICE 'Tabla master.energy_plants ya existe, saltando movimiento';
    ELSE
        RAISE NOTICE 'Tabla public.energy_plants no existe, saltando';
    END IF;
END $$;
-- +goose StatementEnd

-- =============================================================================
-- PASO 2: Crear tabla operational.events_std (PostgreSQL estándar)
-- =============================================================================
CREATE TABLE IF NOT EXISTS operational.events_std (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type varchar(100) NOT NULL,
    plant_source_id uuid NOT NULL REFERENCES master.energy_plants(id),
    source varchar(255),
    data text,
    metadata text,
    created_at timestamptz DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_op_event_type ON operational.events_std(event_type);
CREATE INDEX IF NOT EXISTS idx_op_plant_source_id ON operational.events_std(plant_source_id);
CREATE INDEX IF NOT EXISTS idx_op_created_at ON operational.events_std(created_at DESC);

-- =============================================================================
-- PASO 3: Crear tabla analytical.events_ts
-- =============================================================================
CREATE TABLE IF NOT EXISTS analytical.events_ts (
    created_at timestamptz NOT NULL DEFAULT now(),
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    event_type varchar(100) NOT NULL,
    plant_source_id uuid NOT NULL,
    source varchar(255),
    data text,
    metadata text,
    PRIMARY KEY (created_at, id)
);

CREATE INDEX IF NOT EXISTS idx_an_event_type ON analytical.events_ts(event_type);
CREATE INDEX IF NOT EXISTS idx_an_plant_source_id ON analytical.events_ts(plant_source_id);

-- =============================================================================
-- PASO 3b: Convertir a TimescaleDB hypertable (solo si la extensión YA está instalada)
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
-- PASO 4: Migrar datos existentes (si la tabla events existe)
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
-- PASO 5: Eliminar tabla legacy (opcional - comentado por seguridad)
-- =============================================================================
-- DROP TABLE IF EXISTS public.events;

-- =============================================================================
-- PASO 6: Configurar compresión para hypertable (opcional)
-- =============================================================================
-- ALTER TABLE analytical.events_ts SET (
--     timescaledb.compress,
--     timescaledb.compress_segmentby = 'plant_source_id'
-- );
-- SELECT add_compression_policy('analytical.events_ts', INTERVAL '7 days');

-- +goose Down
-- Revertir migración

-- Recrear tabla events en public si no existe
CREATE TABLE IF NOT EXISTS public.events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type varchar(100) NOT NULL,
    plant_source_id uuid NOT NULL,
    source varchar(255),
    data text,
    metadata text,
    created_at timestamptz DEFAULT now()
);

-- Migrar datos de vuelta a public.events
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'operational' AND table_name = 'events_std') THEN
        INSERT INTO public.events (id, event_type, plant_source_id, source, data, metadata, created_at)
        SELECT id, event_type, plant_source_id, source, data, metadata, created_at
        FROM operational.events_std
        ON CONFLICT (id) DO NOTHING;
    END IF;
END $$;
-- +goose StatementEnd

-- Eliminar tablas de nuevos esquemas
DROP TABLE IF EXISTS analytical.events_ts;
DROP TABLE IF EXISTS operational.events_std;

-- Mover energy_plants de vuelta a public
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'master' AND table_name = 'energy_plants') THEN
        ALTER TABLE master.energy_plants SET SCHEMA public;
    END IF;
END $$;
-- +goose StatementEnd
