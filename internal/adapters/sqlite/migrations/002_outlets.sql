CREATE TABLE outlets (
    id TEXT PRIMARY KEY NOT NULL,
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    address TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
