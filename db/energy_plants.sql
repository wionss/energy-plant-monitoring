-- Insert energy plants into the master schema
-- NOTE: Run after applying the add_geo_and_cache_fields migration
INSERT INTO master.energy_plants (id, plant_name, plant_type, location, latitude, longitude, capacity_mw, created_at) VALUES
('1e2d3c4b-5a6f-7e8d-9c0b-1a2b3c4d5e6f', 'Solar Plant Pasto', 'solar', 'Pasto, Nariño, Colombia', 1.2136, -77.2811, 150.0, now()),
('2f3e4d5c-6b7a-8c9d-0e1f-2b3c4d5e6f7a', 'Wind Farm Cali', 'wind', 'Cali, Valle del Cauca, Colombia', 3.4516, -76.5320, 200.0, now()),
('c2e78b94-76f4-49a4-b6e8-d62c8d1d23ea', 'Hydro Plant Bogotá', 'hydro', 'Bogotá, Cundinamarca, Colombia', 4.7110, -74.0721, 300.0, now())
ON CONFLICT (id) DO UPDATE SET
    plant_name = EXCLUDED.plant_name,
    plant_type = EXCLUDED.plant_type,
    location = EXCLUDED.location,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude,
    capacity_mw = EXCLUDED.capacity_mw;