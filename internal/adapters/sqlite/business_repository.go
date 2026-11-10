package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/satubakul/desktop-pos-community/internal/domain/business"
)

func (s *Store) Create(ctx context.Context, business domain.Business) error {
	if err := domain.Validate(business); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create business: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO businesses (id, name, phone, address, created_at) VALUES (?, ?, ?, ?, ?)`,
		business.ID,
		business.Name,
		business.Phone,
		business.Address,
		formatTime(business.CreatedAt),
	); err != nil {
		return fmt.Errorf("save business: %w", err)
	}

	for _, profile := range business.Profiles {
		primary := 0
		if profile.Primary {
			primary = 1
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO business_profiles (business_id, profile_key, display_name, is_primary) VALUES (?, ?, ?, ?)`,
			business.ID,
			profile.Key,
			profile.Name,
			primary,
		); err != nil {
			return fmt.Errorf("save business profile: %w", err)
		}
	}

	for _, outlet := range business.Outlets {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO outlets (id, business_id, name, address, created_at) VALUES (?, ?, ?, ?, ?)`,
			outlet.ID,
			business.ID,
			outlet.Name,
			outlet.Address,
			formatTime(outlet.CreatedAt),
		); err != nil {
			return fmt.Errorf("save outlet: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO app_state (key, value) VALUES ('active_business_id', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, business.ID); err != nil {
		return fmt.Errorf("set active business: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit business creation: %w", err)
	}
	return nil
}

func (s *Store) List(ctx context.Context) ([]domain.Business, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, phone, address, created_at
		FROM businesses
		ORDER BY name COLLATE NOCASE, id
	`)
	if err != nil {
		return nil, fmt.Errorf("list businesses: %w", err)
	}

	var businesses []domain.Business
	for rows.Next() {
		var item domain.Business
		var createdAt string
		if err := rows.Scan(&item.ID, &item.Name, &item.Phone, &item.Address, &createdAt); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("read business row: %w", err)
		}
		item.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("parse business timestamp: %w", err)
		}
		businesses = append(businesses, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("read businesses: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close business rows: %w", err)
	}

	for i := range businesses {
		businesses[i].Profiles, err = s.listProfiles(ctx, businesses[i].ID)
		if err != nil {
			return nil, err
		}
		businesses[i].Outlets, err = s.listOutlets(ctx, businesses[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return businesses, nil
}

func (s *Store) ActiveID(ctx context.Context) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx,
		`SELECT value FROM app_state WHERE key = 'active_business_id'`,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read active business: %w", err)
	}
	return id, nil
}

func (s *Store) SetActive(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin workspace switch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var found string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM businesses WHERE id = ?`, id).Scan(&found); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("usaha tidak ditemukan")
		}
		return fmt.Errorf("verify business: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO app_state (key, value) VALUES ('active_business_id', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, id); err != nil {
		return fmt.Errorf("save active business: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit workspace switch: %w", err)
	}
	return nil
}

func (s *Store) listProfiles(ctx context.Context, businessID string) ([]domain.Profile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT profile_key, display_name, is_primary
		FROM business_profiles
		WHERE business_id = ?
		ORDER BY is_primary DESC, display_name COLLATE NOCASE
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list profiles for business %s: %w", businessID, err)
	}
	defer rows.Close()

	var profiles []domain.Profile
	for rows.Next() {
		var profile domain.Profile
		var primary int
		if err := rows.Scan(&profile.Key, &profile.Name, &primary); err != nil {
			return nil, fmt.Errorf("read business profile: %w", err)
		}
		profile.Primary = primary == 1
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read business profiles: %w", err)
	}
	return profiles, nil
}

func (s *Store) listOutlets(ctx context.Context, businessID string) ([]domain.Outlet, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, address, created_at
		FROM outlets
		WHERE business_id = ?
		ORDER BY created_at, id
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list outlets for business %s: %w", businessID, err)
	}
	defer rows.Close()

	var outlets []domain.Outlet
	for rows.Next() {
		var outlet domain.Outlet
		var createdAt string
		if err := rows.Scan(&outlet.ID, &outlet.Name, &outlet.Address, &createdAt); err != nil {
			return nil, fmt.Errorf("read outlet row: %w", err)
		}
		outlet.BusinessID = businessID
		outlet.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse outlet timestamp: %w", err)
		}
		outlets = append(outlets, outlet)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read outlets: %w", err)
	}
	return outlets, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}
