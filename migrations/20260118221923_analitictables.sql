-- +goose Up
-- Migration: Analytical tables setup
-- NOTE: The analytical and operational tables were already created in 20260118120000_multi_schema_timescaledb.sql
-- This migration only adds additional indexes if needed

-- Additional indexes to optimize analytical queries (if not already present)
CREATE INDEX IF NOT EXISTS idx_an_created_at ON analytical.events_ts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_op_source ON operational.events_std(source);

-- +goose Down
-- Eliminate aditional Indexes created in the Up migration
DROP INDEX IF EXISTS analytical.idx_an_created_at;
DROP INDEX IF EXISTS operational.idx_op_source;
