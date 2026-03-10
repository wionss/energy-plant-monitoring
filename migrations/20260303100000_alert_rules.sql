-- +goose Up
CREATE TABLE master.alert_rules (
    id              UUID         DEFAULT gen_random_uuid() PRIMARY KEY,
    name            VARCHAR(100) NOT NULL UNIQUE,
    event_type      VARCHAR(100) NOT NULL,
    condition       VARCHAR(20)  NOT NULL,
    field_path      VARCHAR(255) NOT NULL,
    threshold       DOUBLE PRECISION,
    threshold_str   VARCHAR(255),
    severity        VARCHAR(20)  NOT NULL DEFAULT 'warning',
    notification_msg TEXT,
    active          BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ  DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  DEFAULT NOW()
);
-- Seed: reglas por defecto (antes hardcodeadas en getDefaultAlertRules)
INSERT INTO master.alert_rules
    (name, event_type, condition, field_path, threshold, severity, notification_msg)
VALUES
    ('high_temperature',     'temperature', 'gt', 'temperature', 50,  'warning',  'Temperatura muy alta detectada'),
    ('critical_temperature', 'temperature', 'gt', 'temperature', 80,  'critical', '¡Temperatura crítica! Intervención inmediata requerida'),
    ('pressure_critical',    'pressure',    'gt', 'pressure',    100, 'critical', 'Presión crítica detectada'),
    ('power_loss',           'status',      'eq', 'status',      0,   'critical', 'Pérdida de energía en planta');

-- +goose Down
DROP TABLE master.alert_rules;
