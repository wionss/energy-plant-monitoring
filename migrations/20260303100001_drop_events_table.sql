-- +goose Up
DROP INDEX IF EXISTS idx_plant_source_id;
DROP INDEX IF EXISTS idx_event_type;
DROP INDEX IF EXISTS idx_created_at;
DROP TABLE IF EXISTS events;

-- +goose Down
CREATE TABLE events (
    id              UUID    NOT NULL DEFAULT gen_random_uuid(),
    event_type      VARCHAR(100) NOT NULL,
    plant_source_id UUID    NOT NULL,
    source          VARCHAR(255),
    data            JSONB   NOT NULL,
    metadata        JSONB,
    created_at      TIMESTAMPTZ,
    PRIMARY KEY (id),
    CONSTRAINT fk_events_plant_source
        FOREIGN KEY (plant_source_id) REFERENCES master.energy_plants(id)
);
CREATE INDEX idx_created_at      ON events (created_at);
CREATE INDEX idx_event_type      ON events (event_type);
CREATE INDEX idx_plant_source_id ON events (plant_source_id);
