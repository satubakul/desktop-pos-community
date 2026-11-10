CREATE TABLE transaction_journal (
    id TEXT PRIMARY KEY NOT NULL,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    outlet_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    amount_rupiah INTEGER NOT NULL DEFAULT 0 CHECK (amount_rupiah >= 0),
    actor_id TEXT,
    occurred_at TEXT NOT NULL,
    FOREIGN KEY (outlet_id, business_id) REFERENCES outlets(id, business_id)
);
