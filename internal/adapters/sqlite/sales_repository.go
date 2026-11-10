package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	domain "github.com/satubakul/desktop-pos-community/internal/domain/sale"
)

type preparedLine struct {
	itemID          string
	itemName        string
	quantity        int64
	unitPriceRupiah int64
	lineTotalRupiah int64
	fulfillment     string
	tracksStock     bool
}

func (s *Store) Checkout(ctx context.Context, input domain.CheckoutInput, requestHash string) (domain.Receipt, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Receipt{}, fmt.Errorf("begin checkout: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existingID, existingHash string
	err = tx.QueryRowContext(ctx, `
		SELECT id, request_hash FROM sales WHERE business_id = ? AND operation_id = ?
	`, input.BusinessID, input.OperationID).Scan(&existingID, &existingHash)
	if err == nil {
		if existingHash != requestHash {
			return domain.Receipt{}, errors.New("ID transaksi sudah digunakan untuk data yang berbeda")
		}
		receipt, err := readReceipt(ctx, tx, existingID, input.BusinessID)
		if err != nil {
			return domain.Receipt{}, err
		}
		if err := tx.Commit(); err != nil {
			return domain.Receipt{}, fmt.Errorf("finish idempotent checkout: %w", err)
		}
		return receipt, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.Receipt{}, fmt.Errorf("check previous checkout: %w", err)
	}

	var outletID string
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM outlets WHERE id = ? AND business_id = ?
	`, input.OutletID, input.BusinessID).Scan(&outletID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Receipt{}, errors.New("outlet tidak ditemukan pada usaha aktif")
		}
		return domain.Receipt{}, fmt.Errorf("verify checkout outlet: %w", err)
	}

	prepared, total, err := prepareLines(ctx, tx, input)
	if err != nil {
		return domain.Receipt{}, err
	}
	cashReceived := int64(0)
	change := int64(0)
	if input.PaymentMethod == domain.PaymentCash {
		if input.CashReceivedRupiah < total {
			return domain.Receipt{}, fmt.Errorf("uang diterima kurang: total %d, diterima %d", total, input.CashReceivedRupiah)
		}
		cashReceived = input.CashReceivedRupiah
		change = cashReceived - total
	}
	createdAt := time.Now().UTC()
	saleID := input.OperationID
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sales (
			id, business_id, outlet_id, operation_id, request_hash, status,
			total_rupiah, created_at
		) VALUES (?, ?, ?, ?, ?, 'completed', ?, ?)
	`,
		saleID,
		input.BusinessID,
		input.OutletID,
		input.OperationID,
		requestHash,
		total,
		formatTime(createdAt),
	); err != nil {
		return domain.Receipt{}, fmt.Errorf("save sale: %w", err)
	}

	receipt := domain.Receipt{
		ID:                 saleID,
		OperationID:        input.OperationID,
		BusinessID:         input.BusinessID,
		OutletID:           input.OutletID,
		TotalRupiah:        total,
		PaymentMethod:      input.PaymentMethod,
		CashReceivedRupiah: cashReceived,
		ChangeRupiah:       change,
		CreatedAt:          createdAt,
		Lines:              make([]domain.Line, 0, len(prepared)),
	}
	for index, line := range prepared {
		if line.tracksStock && line.fulfillment == "stock" {
			result, err := tx.ExecContext(ctx, `
				UPDATE stock_balances
				SET quantity_on_hand = quantity_on_hand - ?, updated_at = ?
				WHERE business_id = ? AND outlet_id = ? AND item_id = ? AND quantity_on_hand >= ?
			`, line.quantity, formatTime(createdAt), input.BusinessID, input.OutletID, line.itemID, line.quantity)
			if err != nil {
				return domain.Receipt{}, fmt.Errorf("decrease stock for %s: %w", line.itemName, err)
			}
			changed, err := result.RowsAffected()
			if err != nil {
				return domain.Receipt{}, fmt.Errorf("check stock update for %s: %w", line.itemName, err)
			}
			if changed != 1 {
				return domain.Receipt{}, fmt.Errorf("stok %s tidak mencukupi", line.itemName)
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO stock_movements (
					id, business_id, outlet_id, item_id, movement_type,
					quantity_delta, source_type, source_id, created_at
				) VALUES (lower(hex(randomblob(16))), ?, ?, ?, 'sale', ?, 'sale', ?, ?)
			`, input.BusinessID, input.OutletID, line.itemID, -line.quantity, saleID, formatTime(createdAt)); err != nil {
				return domain.Receipt{}, fmt.Errorf("record stock movement for %s: %w", line.itemName, err)
			}
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sale_lines (
				id, business_id, sale_id, item_id, line_number, item_name,
				quantity, unit_price_rupiah, line_total_rupiah, fulfillment_type
			) VALUES (lower(hex(randomblob(16))), ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, input.BusinessID, saleID, line.itemID, index, line.itemName, line.quantity, line.unitPriceRupiah, line.lineTotalRupiah, line.fulfillment); err != nil {
			return domain.Receipt{}, fmt.Errorf("save sale line for %s: %w", line.itemName, err)
		}
		receipt.Lines = append(receipt.Lines, domain.Line{
			ItemID:          line.itemID,
			Name:            line.itemName,
			Quantity:        line.quantity,
			UnitPriceRupiah: line.unitPriceRupiah,
			LineTotalRupiah: line.lineTotalRupiah,
			Fulfillment:     line.fulfillment,
		})
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO payments (
			id, business_id, sale_id, method, amount_rupiah,
			cash_received_rupiah, change_rupiah, created_at
		) VALUES (lower(hex(randomblob(16))), ?, ?, ?, ?, ?, ?, ?)
	`, input.BusinessID, saleID, input.PaymentMethod, total, cashReceived, change, formatTime(createdAt)); err != nil {
		return domain.Receipt{}, fmt.Errorf("save payment: %w", err)
	}
	if err := insertJournalEvent(ctx, tx, input.BusinessID, input.OutletID, "sale", saleID, total, createdAt); err != nil {
		return domain.Receipt{}, err
	}
	if err := insertJournalEvent(ctx, tx, input.BusinessID, input.OutletID, "payment", saleID, total, createdAt); err != nil {
		return domain.Receipt{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Receipt{}, fmt.Errorf("commit checkout: %w", err)
	}
	return receipt, nil
}

func prepareLines(ctx context.Context, tx *sql.Tx, input domain.CheckoutInput) ([]preparedLine, int64, error) {
	lines := make([]preparedLine, 0, len(input.Lines))
	var total int64
	for _, requested := range input.Lines {
		var itemName, itemType string
		var unitPrice int64
		var tracksStock, allowsPreorder, active int
		err := tx.QueryRowContext(ctx, `
			SELECT name, item_type, price_rupiah, tracks_stock, allows_preorder, is_active
			FROM catalog_items WHERE id = ? AND business_id = ?
		`, requested.ItemID, input.BusinessID).Scan(&itemName, &itemType, &unitPrice, &tracksStock, &allowsPreorder, &active)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, fmt.Errorf("item %s tidak ditemukan pada usaha aktif", requested.ItemID)
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read checkout item: %w", err)
		}
		if active != 1 {
			return nil, 0, fmt.Errorf("item %s sudah tidak aktif", itemName)
		}

		line := preparedLine{
			itemID:          requested.ItemID,
			itemName:        itemName,
			quantity:        requested.Quantity,
			unitPriceRupiah: unitPrice,
			tracksStock:     tracksStock == 1,
		}
		switch {
		case requested.PreOrder:
			if line.tracksStock && allowsPreorder != 1 {
				return nil, 0, fmt.Errorf("pre-order belum diaktifkan untuk %s", itemName)
			}
			if !line.tracksStock {
				return nil, 0, fmt.Errorf("item %s tidak mendukung pre-order", itemName)
			}
			line.fulfillment = "preorder"
		case line.tracksStock:
			var available int64
			err := tx.QueryRowContext(ctx, `
				SELECT quantity_on_hand FROM stock_balances
				WHERE business_id = ? AND outlet_id = ? AND item_id = ?
			`, input.BusinessID, input.OutletID, requested.ItemID).Scan(&available)
			if errors.Is(err, sql.ErrNoRows) {
				available = 0
			} else if err != nil {
				return nil, 0, fmt.Errorf("read stock for %s: %w", itemName, err)
			}
			if available < requested.Quantity {
				return nil, 0, fmt.Errorf("stok %s tidak cukup: tersedia %d, diminta %d; tandai sebagai pre-order jika sesuai", itemName, available, requested.Quantity)
			}
			line.fulfillment = "stock"
		default:
			line.fulfillment = "nonstock"
		}

		if unitPrice > 0 && requested.Quantity > math.MaxInt64/unitPrice {
			return nil, 0, errors.New("nilai transaksi terlalu besar")
		}
		line.lineTotalRupiah = unitPrice * requested.Quantity
		if total > math.MaxInt64-line.lineTotalRupiah {
			return nil, 0, errors.New("nilai transaksi terlalu besar")
		}
		total += line.lineTotalRupiah
		lines = append(lines, line)
	}
	return lines, total, nil
}

func insertJournalEvent(ctx context.Context, tx *sql.Tx, businessID, outletID, eventType, saleID string, amount int64, occurredAt time.Time) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO transaction_journal (
			id, business_id, outlet_id, event_type, source_type, source_id, amount_rupiah, occurred_at
		) VALUES (lower(hex(randomblob(16))), ?, ?, ?, 'sale', ?, ?, ?)
	`, businessID, outletID, eventType, saleID, amount, formatTime(occurredAt)); err != nil {
		return fmt.Errorf("record %s journal event: %w", eventType, err)
	}
	return nil
}

func readReceipt(ctx context.Context, tx *sql.Tx, saleID, businessID string) (domain.Receipt, error) {
	var receipt domain.Receipt
	var createdAt string
	if err := tx.QueryRowContext(ctx, `
		SELECT s.id, s.operation_id, s.business_id, s.outlet_id, s.total_rupiah,
			p.method, p.cash_received_rupiah, p.change_rupiah, s.created_at
		FROM sales AS s
		JOIN payments AS p ON p.sale_id = s.id AND p.business_id = s.business_id
		WHERE s.id = ? AND s.business_id = ?
	`, saleID, businessID).Scan(
		&receipt.ID,
		&receipt.OperationID,
		&receipt.BusinessID,
		&receipt.OutletID,
		&receipt.TotalRupiah,
		&receipt.PaymentMethod,
		&receipt.CashReceivedRupiah,
		&receipt.ChangeRupiah,
		&createdAt,
	); err != nil {
		return domain.Receipt{}, fmt.Errorf("read saved receipt: %w", err)
	}
	var err error
	receipt.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return domain.Receipt{}, fmt.Errorf("parse receipt time: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT item_id, item_name, quantity, unit_price_rupiah, line_total_rupiah, fulfillment_type
		FROM sale_lines WHERE sale_id = ? AND business_id = ? ORDER BY line_number
	`, saleID, businessID)
	if err != nil {
		return domain.Receipt{}, fmt.Errorf("read receipt lines: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var line domain.Line
		if err := rows.Scan(&line.ItemID, &line.Name, &line.Quantity, &line.UnitPriceRupiah, &line.LineTotalRupiah, &line.Fulfillment); err != nil {
			return domain.Receipt{}, fmt.Errorf("read receipt line: %w", err)
		}
		receipt.Lines = append(receipt.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return domain.Receipt{}, fmt.Errorf("read receipt lines: %w", err)
	}
	return receipt, nil
}

func (s *Store) ListJournal(ctx context.Context, businessID string) ([]domain.JournalEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, business_id, outlet_id, event_type, source_type, source_id, amount_rupiah, occurred_at
		FROM transaction_journal
		WHERE business_id = ?
		ORDER BY occurred_at DESC, id DESC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list transaction journal: %w", err)
	}
	defer rows.Close()

	var entries []domain.JournalEntry
	for rows.Next() {
		var entry domain.JournalEntry
		var occurredAt string
		if err := rows.Scan(&entry.ID, &entry.BusinessID, &entry.OutletID, &entry.EventType, &entry.SourceType, &entry.SourceID, &entry.AmountRupiah, &occurredAt); err != nil {
			return nil, fmt.Errorf("read journal entry: %w", err)
		}
		entry.OccurredAt, err = parseTime(occurredAt)
		if err != nil {
			return nil, fmt.Errorf("parse journal time: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read transaction journal: %w", err)
	}
	return entries, nil
}
