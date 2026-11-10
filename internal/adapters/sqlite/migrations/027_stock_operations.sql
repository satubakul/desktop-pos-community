CREATE TABLE stock_operations (
    id TEXT PRIMARY KEY NOT NULL,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    outlet_id TEXT NOT NULL,
    item_id TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    operation_type TEXT NOT NULL CHECK (operation_type IN ('restock', 'adjustment')),
    quantity_delta INTEGER NOT NULL CHECK (quantity_delta <> 0),
    quantity_after INTEGER NOT NULL CHECK (quantity_after >= 0),
    reason TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (business_id, operation_id),
    CHECK ((operation_type = 'restock' AND quantity_delta > 0) OR operation_type = 'adjustment'),
    CHECK (length(trim(reason)) > 0),
    FOREIGN KEY (outlet_id, business_id) REFERENCES outlets(id, business_id),
    FOREIGN KEY (item_id, business_id) REFERENCES catalog_items(id, business_id)
);
