CREATE TABLE payments (
    id TEXT PRIMARY KEY NOT NULL,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    sale_id TEXT NOT NULL,
    method TEXT NOT NULL CHECK (method = 'cash'),
    amount_rupiah INTEGER NOT NULL CHECK (amount_rupiah >= 0),
    created_at TEXT NOT NULL,
    FOREIGN KEY (sale_id, business_id) REFERENCES sales(id, business_id)
);
