EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM operational.events_std
WHERE event_type = 'alert';

-- Tabla TimescaleDB (Analytical)
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM analytical.events_ts
WHERE event_type = 'alert';


-- Tabla Standard (Operational)
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM operational.events_std
WHERE event_type = 'alert'
AND created_at > NOW() - INTERVAL '7 days';
-- 1815  49
-- Tabla TimescaleDB (Analytical)
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM analytical.events_ts
WHERE event_type = 'alert'
AND created_at > NOW() - INTERVAL '7 days';
-- 1792  40

-- Tabla Standard (Operational) - Group by manual costoso
EXPLAIN (ANALYZE, BUFFERS)
SELECT date_trunc('hour', created_at) as bucket, count(*)
FROM operational.events_std
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY 1;

-- Tabla TimescaleDB (Analytical) - Optimizado
EXPLAIN (ANALYZE, BUFFERS)
SELECT time_bucket('1 hour', created_at) as bucket, count(*)
FROM analytical.events_ts
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY 1;