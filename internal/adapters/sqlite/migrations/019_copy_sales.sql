INSERT INTO sales (id, business_id, outlet_id, operation_id, request_hash, status, total_rupiah, created_at)
SELECT id, business_id, outlet_id, operation_id, request_hash, status, total_rupiah, created_at
FROM sales_legacy;
