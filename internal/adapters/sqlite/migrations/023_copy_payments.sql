INSERT INTO payments (id, business_id, sale_id, method, amount_rupiah, cash_received_rupiah, change_rupiah, created_at)
SELECT p.id, p.business_id, p.sale_id, p.method, p.amount_rupiah, s.cash_received_rupiah, s.change_rupiah, p.created_at
FROM payments_legacy AS p
JOIN sales_legacy AS s ON s.id = p.sale_id AND s.business_id = p.business_id;
