-- +goose Up
-- Migration: Analytical tables setup
-- NOTA: Las tablas analytical y operational ya fueron creadas en 20260118120000_multi_schema_timescaledb.sql
-- Esta migración solo añade índices adicionales si son necesarios

-- Índices adicionales para optimización de consultas analíticas (si no existen)
CREATE INDEX IF NOT EXISTS idx_an_created_at ON analytical.events_ts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_op_source ON operational.events_std(source);

-- +goose Down
-- Eliminar índices adicionales
DROP INDEX IF EXISTS analytical.idx_an_created_at;
DROP INDEX IF EXISTS operational.idx_op_source;
