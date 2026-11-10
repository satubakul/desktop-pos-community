CREATE TABLE catalog_item_assets (
    id TEXT PRIMARY KEY NOT NULL,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    item_id TEXT NOT NULL,
    asset_type TEXT NOT NULL CHECK (asset_type = 'image'),
    file_name TEXT NOT NULL,
    content_type TEXT NOT NULL CHECK (content_type IN ('image/jpeg', 'image/png', 'image/gif')),
    data BLOB NOT NULL CHECK (length(data) > 0),
    created_at TEXT NOT NULL,
    UNIQUE (business_id, item_id, asset_type),
    FOREIGN KEY (item_id, business_id) REFERENCES catalog_items(id, business_id)
);
