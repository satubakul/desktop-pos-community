CREATE TABLE stock_balances (
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    outlet_id TEXT NOT NULL,
    item_id TEXT NOT NULL,
    quantity_on_hand INTEGER NOT NULL DEFAULT 0 CHECK (quantity_on_hand >= 0),
    updated_at TEXT NOT NULL,
    PRIMARY KEY (business_id, outlet_id, item_id),
    FOREIGN KEY (outlet_id, business_id) REFERENCES outlets(id, business_id),
    FOREIGN KEY (item_id, business_id) REFERENCES catalog_items(id, business_id)
);
