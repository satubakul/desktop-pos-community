CREATE TABLE business_profiles (
    business_id TEXT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    profile_key TEXT NOT NULL,
    display_name TEXT NOT NULL CHECK (length(trim(display_name)) > 0),
    is_primary INTEGER NOT NULL DEFAULT 0 CHECK (is_primary IN (0, 1)),
    PRIMARY KEY (business_id, profile_key)
);
