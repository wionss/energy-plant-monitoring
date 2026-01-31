-- +goose Up
-- create "energy_plants" table in master schema
CREATE TABLE "master"."energy_plants" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "plant_name" character varying(255) NOT NULL,
  "location" character varying(255) NULL,
  "capacity_mw" numeric NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- create index "idx_energy_plants_deleted_at" to table: "energy_plants"
CREATE INDEX "idx_master_energy_plants_deleted_at" ON "master"."energy_plants" ("deleted_at");
-- create "examples" table
CREATE TABLE "examples" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(255) NOT NULL,
  "description" text NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- create "events" table
CREATE TABLE "events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "event_type" character varying(100) NOT NULL,
  "plant_source_id" uuid NOT NULL,
  "source" character varying(255) NULL,
  "data" jsonb NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_events_plant_source" FOREIGN KEY ("plant_source_id") REFERENCES "master"."energy_plants" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "idx_created_at" to table: "events"
CREATE INDEX "idx_created_at" ON "events" ("created_at");
-- create index "idx_event_type" to table: "events"
CREATE INDEX "idx_event_type" ON "events" ("event_type");
-- create index "idx_plant_source_id" to table: "events"
CREATE INDEX "idx_plant_source_id" ON "events" ("plant_source_id");

-- +goose Down
-- reverse: create index "idx_plant_source_id" to table: "events"
DROP INDEX "idx_plant_source_id";
-- reverse: create index "idx_event_type" to table: "events"
DROP INDEX "idx_event_type";
-- reverse: create index "idx_created_at" to table: "events"
DROP INDEX "idx_created_at";
-- reverse: create "events" table
DROP TABLE "events";
-- reverse: create "examples" table
DROP TABLE "examples";
-- reverse: create index "idx_master_energy_plants_deleted_at" to table: "energy_plants"
DROP INDEX "master"."idx_master_energy_plants_deleted_at";
-- reverse: create "energy_plants" table
DROP TABLE "master"."energy_plants";
