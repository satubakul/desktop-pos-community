CREATE UNIQUE INDEX catalog_items_id_business_unique
    ON catalog_items (id, business_id);
