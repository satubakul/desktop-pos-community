CREATE TABLE sale_lines (
    id TEXT PRIMARY KEY NOT NULL,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    sale_id TEXT NOT NULL,
    item_id TEXT NOT NULL,
    line_number INTEGER NOT NULL CHECK (line_number >= 0),
    item_name TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_rupiah INTEGER NOT NULL CHECK (unit_price_rupiah >= 0),
    line_total_rupiah INTEGER NOT NULL CHECK (line_total_rupiah = quantity * unit_price_rupiah),
    fulfillment_type TEXT NOT NULL CHECK (fulfillment_type IN ('stock', 'preorder', 'nonstock')),
    FOREIGN KEY (sale_id, business_id) REFERENCES sales(id, business_id),
    FOREIGN KEY (item_id, business_id) REFERENCES catalog_items(id, business_id)
);
