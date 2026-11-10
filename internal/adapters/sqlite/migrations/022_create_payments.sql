CREATE TABLE payments (
    id TEXT PRIMARY KEY NOT NULL,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    sale_id TEXT NOT NULL,
    method TEXT NOT NULL CHECK (method IN ('cash', 'bank_transfer', 'card', 'e_wallet')),
    amount_rupiah INTEGER NOT NULL CHECK (amount_rupiah >= 0),
    cash_received_rupiah INTEGER NOT NULL CHECK (cash_received_rupiah >= 0),
    change_rupiah INTEGER NOT NULL CHECK (change_rupiah >= 0),
    created_at TEXT NOT NULL,
    UNIQUE (sale_id, business_id),
    CHECK (
        (method = 'cash' AND cash_received_rupiah >= amount_rupiah AND change_rupiah = cash_received_rupiah - amount_rupiah)
        OR (method <> 'cash' AND cash_received_rupiah = 0 AND change_rupiah = 0)
    ),
    FOREIGN KEY (sale_id, business_id) REFERENCES sales(id, business_id)
);
