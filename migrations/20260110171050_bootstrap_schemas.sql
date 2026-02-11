-- +goose Up
-- Bootstrap: Create schemas before other migrations
-- This migration MUST run before any table creation that uses these schemas

-- =============================================================================
-- Crear esquemas para arquitectura multi-schema
-- =============================================================================
CREATE SCHEMA IF NOT EXISTS master;
CREATE SCHEMA IF NOT EXISTS operational;
CREATE SCHEMA IF NOT EXISTS analytical;

-- +goose Down
-- No eliminamos esquemas en down para evitar pérdida de datos accidental
-- Si necesitas eliminarlos, hazlo manualmente
