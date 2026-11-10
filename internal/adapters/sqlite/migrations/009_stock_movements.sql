CREATE TABLE stock_movements (
    id TEXT PRIMARY KEY NOT NULL,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    outlet_id TEXT NOT NULL,
    item_id TEXT NOT NULL,
    movement_type TEXT NOT NULL,
    quantity_delta INTEGER NOT NULL CHECK (quantity_delta <> 0),
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (outlet_id, business_id) REFERENCES outlets(id, business_id),
    FOREIGN KEY (item_id, business_id) REFERENCES catalog_items(id, business_id)
);
