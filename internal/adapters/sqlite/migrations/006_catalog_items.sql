CREATE TABLE catalog_items (
    id TEXT PRIMARY KEY NOT NULL,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    sku TEXT NOT NULL DEFAULT '',
    barcode TEXT NOT NULL DEFAULT '',
    unit TEXT NOT NULL DEFAULT 'pcs',
    item_type TEXT NOT NULL CHECK (item_type IN ('good', 'service')),
    price_rupiah INTEGER NOT NULL CHECK (price_rupiah >= 0),
    tracks_stock INTEGER NOT NULL DEFAULT 0 CHECK (tracks_stock IN (0, 1)),
    allows_preorder INTEGER NOT NULL DEFAULT 0 CHECK (allows_preorder IN (0, 1)),
    is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    created_at TEXT NOT NULL,
    CHECK (allows_preorder = 0 OR tracks_stock = 1)
);
