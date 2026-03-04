-- +goose Up
-- La política de retención SOLO funciona en hypertables.
-- Aplicamos la retención a la tabla analítica (por ejemplo, borrar datos con más de 90 o 30 días)
SELECT add_retention_policy(
    'analytical.events_ts',
    INTERVAL '90 days',
    if_not_exists => true
);

-- Si aplicaste la "Opción 1" de la respuesta anterior y creaste el continuous aggregate,
-- también puedes ponerle política de retención a esa vista (opcional):
-- SELECT add_retention_policy('analytical.events_hourly_agg', INTERVAL '180 days', if_not_exists => true);

-- +goose Down
SELECT remove_retention_policy('analytical.events_ts', if_exists => true);
