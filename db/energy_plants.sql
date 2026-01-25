-- Insertar plantas de energía en el esquema master
-- NOTA: Ejecutar después de aplicar la migración multi_schema_timescaledb
insert into master.energy_plants (id, plant_name , location, capacity_mw, created_at) values
('1e2d3c4b-5a6f-7e8d-9c0b-1a2b3c4d5e6f', 'Solar Plant Alpha', 'California, USA', 150.0, now()),
('2f3e4d5c-6b7a-8c9d-0e1f-2b3c4d5e6f7a', 'Wind Farm Beta', 'Texas, USA', 200.0, now()),
('c2e78b94-76f4-49a4-b6e8-d62c8d1d23ea', 'Hydro Plant Gamma', 'Oregon, USA', 300.0, now())
ON CONFLICT (id) DO NOTHING;