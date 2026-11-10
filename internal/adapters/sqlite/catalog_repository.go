package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/satubakul/desktop-pos-community/internal/domain/catalog"
)

func (s *Store) CreateItem(ctx context.Context, item domain.Item, outletID string, initialStock int64, asset *domain.ImageAsset) error {
	if err := domain.Validate(item, initialStock); err != nil {
		return err
	}
	createdAt := item.CreatedAt.UTC()
	if item.CreatedAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create catalog item: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var found string
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM outlets WHERE id = ? AND business_id = ?`,
		outletID,
		item.BusinessID,
	).Scan(&found); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("outlet tidak ditemukan pada usaha aktif")
		}
		return fmt.Errorf("verify catalog outlet: %w", err)
	}

	tracksStock := 0
	if item.TracksStock {
		tracksStock = 1
	}
	allowsPreorder := 0
	if item.AllowsPreorder {
		allowsPreorder = 1
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO catalog_items (
			id, business_id, name, description, category, sku, barcode, unit,
			item_type, price_rupiah, tracks_stock, allows_preorder, is_active, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?)
	`,
		item.ID,
		item.BusinessID,
		item.Name,
		item.Description,
		item.Category,
		item.SKU,
		item.Barcode,
		item.Unit,
		item.Type,
		item.PriceRupiah,
		tracksStock,
		allowsPreorder,
		formatTime(createdAt),
	); err != nil {
		return fmt.Errorf("save catalog item: %w", err)
	}
	if asset != nil {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO catalog_item_assets (
				id, business_id, item_id, asset_type, file_name, content_type, data, created_at
			) VALUES (lower(hex(randomblob(16))), ?, ?, 'image', ?, ?, ?, ?)
		`, item.BusinessID, item.ID, asset.FileName, asset.ContentType, asset.Data, formatTime(createdAt)); err != nil {
			return fmt.Errorf("save catalog item image: %w", err)
		}
	}

	if item.TracksStock {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO stock_balances (business_id, outlet_id, item_id, quantity_on_hand, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, item.BusinessID, outletID, item.ID, initialStock, formatTime(createdAt)); err != nil {
			return fmt.Errorf("create item stock balance: %w", err)
		}
		if initialStock > 0 {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO stock_movements (
					id, business_id, outlet_id, item_id, movement_type,
					quantity_delta, source_type, source_id, created_at
				) VALUES (lower(hex(randomblob(16))), ?, ?, ?, 'opening', ?, 'catalog_item', ?, ?)
			`, item.BusinessID, outletID, item.ID, initialStock, item.ID, formatTime(createdAt)); err != nil {
				return fmt.Errorf("record opening stock movement: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO transaction_journal (
					id, business_id, outlet_id, event_type, source_type, source_id, amount_rupiah, occurred_at
				) VALUES (lower(hex(randomblob(16))), ?, ?, 'stock_opening', 'catalog_item', ?, 0, ?)
			`, item.BusinessID, outletID, item.ID, formatTime(createdAt)); err != nil {
				return fmt.Errorf("record opening stock journal event: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog item: %w", err)
	}
	return nil
}

func (s *Store) ListItems(ctx context.Context, businessID, outletID string) ([]domain.Item, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ci.id, ci.business_id, ci.name, ci.description, ci.category,
			ci.sku, ci.barcode, ci.unit, ci.item_type, ci.price_rupiah,
			ci.tracks_stock, ci.allows_preorder, ci.is_active,
			COALESCE(sb.quantity_on_hand, 0), ci.created_at, COALESCE(ia.file_name, '')
		FROM catalog_items AS ci
	LEFT JOIN stock_balances AS sb
		ON sb.business_id = ci.business_id AND sb.outlet_id = ? AND sb.item_id = ci.id
	LEFT JOIN catalog_item_assets AS ia
		ON ia.business_id = ci.business_id AND ia.item_id = ci.id AND ia.asset_type = 'image'
		WHERE ci.business_id = ? AND ci.is_active = 1
		ORDER BY ci.name COLLATE NOCASE, ci.id
	`, outletID, businessID)
	if err != nil {
		return nil, fmt.Errorf("list catalog items: %w", err)
	}
	defer rows.Close()

	var items []domain.Item
	for rows.Next() {
		var item domain.Item
		var tracksStock, allowsPreorder, active int
		var createdAt, imageFilename string
		if err := rows.Scan(
			&item.ID,
			&item.BusinessID,
			&item.Name,
			&item.Description,
			&item.Category,
			&item.SKU,
			&item.Barcode,
			&item.Unit,
			&item.Type,
			&item.PriceRupiah,
			&tracksStock,
			&allowsPreorder,
			&active,
			&item.StockOnHand,
			&createdAt,
			&imageFilename,
		); err != nil {
			return nil, fmt.Errorf("read catalog item: %w", err)
		}
		item.TracksStock = tracksStock == 1
		item.AllowsPreorder = allowsPreorder == 1
		item.Active = active == 1
		item.ImageFilename = imageFilename
		item.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse catalog item timestamp: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read catalog items: %w", err)
	}
	return items, nil
}

func (s *Store) GetItemImage(ctx context.Context, businessID, itemID string) (domain.ImageAsset, error) {
	var asset domain.ImageAsset
	err := s.db.QueryRowContext(ctx, `
		SELECT file_name, content_type, data
		FROM catalog_item_assets
		WHERE business_id = ? AND item_id = ? AND asset_type = 'image'
	`, businessID, itemID).Scan(&asset.FileName, &asset.ContentType, &asset.Data)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ImageAsset{}, errors.New("gambar item tidak ditemukan")
	}
	if err != nil {
		return domain.ImageAsset{}, fmt.Errorf("read catalog item image: %w", err)
	}
	return asset, nil
}
