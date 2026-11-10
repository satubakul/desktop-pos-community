package business

import (
	"errors"
	"strings"
	"time"
)

type Profile struct {
	Key     string
	Name    string
	Primary bool
}

type Outlet struct {
	ID         string
	BusinessID string
	Name       string
	Address    string
	CreatedAt  time.Time
}

type Business struct {
	ID        string
	Name      string
	Phone     string
	Address   string
	CreatedAt time.Time
	Profiles  []Profile
	Outlets   []Outlet
}

type ProfileOption struct {
	Key  string
	Name string
}

func ProfileOptions() []ProfileOption {
	return []ProfileOption{
		{Key: "computer_electronics", Name: "Komputer & elektronik"},
		{Key: "food_beverage", Name: "F&B / restoran"},
		{Key: "grocery_supermarket", Name: "Swalayan / toko kelontong"},
		{Key: "salon_spa", Name: "Salon / spa"},
		{Key: "workshop_repair", Name: "Bengkel / servis"},
		{Key: "other", Name: "Usaha lainnya"},
	}
}

func Validate(b Business) error {
	if strings.TrimSpace(b.ID) == "" {
		return errors.New("ID usaha wajib diisi")
	}
	if strings.TrimSpace(b.Name) == "" {
		return errors.New("nama usaha wajib diisi")
	}
	if len(b.Profiles) == 0 {
		return errors.New("pilih minimal satu profil usaha")
	}
	if len(b.Outlets) == 0 {
		return errors.New("usaha harus memiliki minimal satu outlet")
	}

	profileKeys := make(map[string]struct{}, len(b.Profiles))
	primaryCount := 0
	for _, profile := range b.Profiles {
		key := strings.TrimSpace(profile.Key)
		if key == "" || strings.TrimSpace(profile.Name) == "" {
			return errors.New("profil usaha tidak valid")
		}
		if _, exists := profileKeys[key]; exists {
			return errors.New("profil usaha tidak boleh duplikat")
		}
		profileKeys[key] = struct{}{}
		if profile.Primary {
			primaryCount++
		}
	}
	if primaryCount != 1 {
		return errors.New("pilih tepat satu profil utama")
	}

	outletIDs := make(map[string]struct{}, len(b.Outlets))
	for _, outlet := range b.Outlets {
		if strings.TrimSpace(outlet.ID) == "" || strings.TrimSpace(outlet.Name) == "" {
			return errors.New("nama outlet wajib diisi")
		}
		if outlet.BusinessID != b.ID {
			return errors.New("outlet harus dimiliki usaha yang sama")
		}
		if _, exists := outletIDs[outlet.ID]; exists {
			return errors.New("ID outlet tidak boleh duplikat")
		}
		outletIDs[outlet.ID] = struct{}{}
	}

	return nil
}
