CREATE TABLE sales (
    id TEXT PRIMARY KEY NOT NULL,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    outlet_id TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status = 'completed'),
    total_rupiah INTEGER NOT NULL CHECK (total_rupiah >= 0),
    cash_received_rupiah INTEGER NOT NULL CHECK (cash_received_rupiah >= total_rupiah),
    change_rupiah INTEGER NOT NULL CHECK (change_rupiah = cash_received_rupiah - total_rupiah),
    created_at TEXT NOT NULL,
    UNIQUE (business_id, operation_id),
    UNIQUE (id, business_id),
    FOREIGN KEY (outlet_id, business_id) REFERENCES outlets(id, business_id)
);
