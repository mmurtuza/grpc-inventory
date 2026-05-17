-- Seed initial inventory data.
-- Executed automatically by the Postgres Docker image on first start
-- (mounted as /docker-entrypoint-initdb.d/02-seed.sql).

INSERT INTO inventory (sku, medicine_name, quantity, requires_prescription) VALUES
    ('MED-001', 'Amoxicillin 500mg',  1450, true),
    ('MED-045', 'Metformin 500mg',     320, true),
    ('MED-089', 'Ibuprofen 200mg',      12, false),
    ('MED-102', 'Lisinopril 10mg',       5, true),
    ('MED-201', 'Cetirizine 10mg',       8, false)
ON CONFLICT (sku) DO NOTHING;
