-- +goose Up
-- Digital Twin: Real-time plant status tracking

CREATE TABLE IF NOT EXISTS master.plant_current_status (
    plant_id uuid PRIMARY KEY REFERENCES master.energy_plants(id),
    last_event_data jsonb NOT NULL,
    current_status varchar(50) NOT NULL DEFAULT 'UNKNOWN',
    last_event_type varchar(100),
    last_event_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Index for filtering by status
CREATE INDEX idx_pcs_status ON master.plant_current_status(current_status);

-- Index for ordering by last update time
CREATE INDEX idx_pcs_updated_at ON master.plant_current_status(updated_at DESC);

-- +goose Down
DROP TABLE IF EXISTS master.plant_current_status;
