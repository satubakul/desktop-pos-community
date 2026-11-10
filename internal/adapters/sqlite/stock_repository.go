package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	domain "github.com/satubakul/desktop-pos-community/internal/domain/stock"
)

func (s *Store) Record(ctx context.Context, operation domain.Operation, requestHash string) (domain.Result, error) {
	if err := domain.Validate(operation); err != nil {
		return domain.Result{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Result{}, fmt.Errorf("begin stock operation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existingHash string
	var existingQuantity int64
	err = tx.QueryRowContext(ctx, `
		SELECT request_hash, quantity_after FROM stock_operations
		WHERE business_id = ? AND operation_id = ?
	`, operation.BusinessID, operation.OperationID).Scan(&existingHash, &existingQuantity)
	if err == nil {
		if existingHash != requestHash {
			return domain.Result{}, errors.New("ID operasi stok sudah digunakan untuk data yang berbeda")
		}
		if err := tx.Commit(); err != nil {
			return domain.Result{}, fmt.Errorf("finish idempotent stock operation: %w", err)
		}
		return domain.Result{OperationID: operation.OperationID, QuantityAfter: existingQuantity}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.Result{}, fmt.Errorf("check previous stock operation: %w", err)
	}

	var tracksStock, active int
	if err := tx.QueryRowContext(ctx, `
		SELECT tracks_stock, is_active FROM catalog_items WHERE id = ? AND business_id = ?
	`, operation.ItemID, operation.BusinessID).Scan(&tracksStock, &active); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Result{}, errors.New("item tidak ditemukan pada usaha aktif")
		}
		return domain.Result{}, fmt.Errorf("verify stock item: %w", err)
	}
	if tracksStock != 1 {
		return domain.Result{}, errors.New("item ini tidak melacak stok")
	}
	if active != 1 {
		return domain.Result{}, errors.New("item ini sudah tidak aktif")
	}

	var quantityBefore int64
	if err := tx.QueryRowContext(ctx, `
		SELECT quantity_on_hand FROM stock_balances
		WHERE business_id = ? AND outlet_id = ? AND item_id = ?
	`, operation.BusinessID, operation.OutletID, operation.ItemID).Scan(&quantityBefore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Result{}, errors.New("saldo stok item tidak ditemukan")
		}
		return domain.Result{}, fmt.Errorf("read stock balance: %w", err)
	}
	if operation.QuantityDelta > 0 && quantityBefore > math.MaxInt64-operation.QuantityDelta {
		return domain.Result{}, errors.New("jumlah stok melebihi batas yang didukung")
	}
	quantityAfter := quantityBefore + operation.QuantityDelta
	if quantityAfter < 0 {
		return domain.Result{}, fmt.Errorf("stok tidak boleh minus; tersedia %d", quantityBefore)
	}

	createdAt := time.Now().UTC()
	update, err := tx.ExecContext(ctx, `
		UPDATE stock_balances SET quantity_on_hand = ?, updated_at = ?
		WHERE business_id = ? AND outlet_id = ? AND item_id = ? AND quantity_on_hand = ?
	`, quantityAfter, formatTime(createdAt), operation.BusinessID, operation.OutletID, operation.ItemID, quantityBefore)
	if err != nil {
		return domain.Result{}, fmt.Errorf("update stock balance: %w", err)
	}
	updated, err := update.RowsAffected()
	if err != nil {
		return domain.Result{}, fmt.Errorf("check stock update: %w", err)
	}
	if updated != 1 {
		return domain.Result{}, errors.New("saldo stok berubah; muat ulang lalu coba lagi")
	}

	var operationID string
	if err := tx.QueryRowContext(ctx, `SELECT lower(hex(randomblob(16)))`).Scan(&operationID); err != nil {
		return domain.Result{}, fmt.Errorf("generate stock operation record ID: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO stock_operations (
			id, business_id, outlet_id, item_id, operation_id, request_hash,
			operation_type, quantity_delta, quantity_after, reason, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, operationID, operation.BusinessID, operation.OutletID, operation.ItemID, operation.OperationID, requestHash,
		operation.Type, operation.QuantityDelta, quantityAfter, operation.Reason, formatTime(createdAt)); err != nil {
		return domain.Result{}, fmt.Errorf("save stock operation: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO stock_movements (
			id, business_id, outlet_id, item_id, movement_type, quantity_delta, source_type, source_id, created_at
		) VALUES (lower(hex(randomblob(16))), ?, ?, ?, ?, ?, 'stock_operation', ?, ?)
	`, operation.BusinessID, operation.OutletID, operation.ItemID, operation.Type, operation.QuantityDelta, operation.OperationID, formatTime(createdAt)); err != nil {
		return domain.Result{}, fmt.Errorf("record stock movement: %w", err)
	}
	eventType := "stock_adjustment"
	if operation.Type == domain.OperationRestock {
		eventType = "stock_restock"
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO transaction_journal (
			id, business_id, outlet_id, event_type, source_type, source_id, amount_rupiah, occurred_at
		) VALUES (lower(hex(randomblob(16))), ?, ?, ?, 'stock_operation', ?, 0, ?)
	`, operation.BusinessID, operation.OutletID, eventType, operation.OperationID, formatTime(createdAt)); err != nil {
		return domain.Result{}, fmt.Errorf("record stock journal event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Result{}, fmt.Errorf("commit stock operation: %w", err)
	}
	return domain.Result{OperationID: operation.OperationID, QuantityAfter: quantityAfter}, nil
}
