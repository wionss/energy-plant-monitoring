-- +goose Up
-- 1. Enriquecer la tabla maestra de plantas
ALTER TABLE "master"."energy_plants"
ADD COLUMN IF NOT EXISTS "plant_type" varchar(50) DEFAULT 'solar',
ADD COLUMN IF NOT EXISTS "latitude" double precision,
ADD COLUMN IF NOT EXISTS "longitude" double precision;

-- 2. Desnormalizar en la tabla analítica (stats) para reportes rápidos
ALTER TABLE "analytical"."hourly_plant_stats"
ADD COLUMN IF NOT EXISTS "plant_name" varchar(255),
ADD COLUMN IF NOT EXISTS "plant_type" varchar(50),
ADD COLUMN IF NOT EXISTS "latitude" double precision,
ADD COLUMN IF NOT EXISTS "longitude" double precision;

-- 3. Crear índice para filtrar por tipo de planta
CREATE INDEX IF NOT EXISTS "idx_hps_plant_type" ON "analytical"."hourly_plant_stats" ("plant_type");

-- +goose Down
ALTER TABLE "analytical"."hourly_plant_stats"
DROP COLUMN IF EXISTS "plant_name",
DROP COLUMN IF EXISTS "plant_type",
DROP COLUMN IF EXISTS "latitude",
DROP COLUMN IF EXISTS "longitude";

ALTER TABLE "master"."energy_plants"
DROP COLUMN IF EXISTS "plant_type",
DROP COLUMN IF EXISTS "latitude",
DROP COLUMN IF EXISTS "longitude";
