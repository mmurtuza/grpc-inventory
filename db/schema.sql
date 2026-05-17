CREATE TABLE IF NOT EXISTS inventory (
    sku                   TEXT PRIMARY KEY,
    medicine_name         TEXT NOT NULL,
    quantity              INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    requires_prescription BOOLEAN NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
