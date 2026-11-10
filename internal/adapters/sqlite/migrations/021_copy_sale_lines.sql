INSERT INTO sale_lines (id, business_id, sale_id, item_id, line_number, item_name, quantity, unit_price_rupiah, line_total_rupiah, fulfillment_type)
SELECT id, business_id, sale_id, item_id, line_number, item_name, quantity, unit_price_rupiah, line_total_rupiah, fulfillment_type
FROM sale_lines_legacy;
