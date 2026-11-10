package catalog

import (
	"errors"
	"strings"
	"time"
)

const (
	TypeGood     = "good"
	TypeService  = "service"
	MaxImageSize = 10 << 20
)

type ImageAsset struct {
	FileName    string
	ContentType string
	Data        []byte
}

type Item struct {
	ID             string
	BusinessID     string
	Name           string
	Description    string
	Category       string
	SKU            string
	Barcode        string
	Unit           string
	Type           string
	PriceRupiah    int64
	TracksStock    bool
	AllowsPreorder bool
	Active         bool
	StockOnHand    int64
	ImageFilename  string
	CreatedAt      time.Time
}

func Validate(item Item, initialStock int64) error {
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.BusinessID) == "" {
		return errors.New("ID barang dan usaha wajib diisi")
	}
	if strings.TrimSpace(item.Name) == "" {
		return errors.New("nama barang atau jasa wajib diisi")
	}
	if item.Type != TypeGood && item.Type != TypeService {
		return errors.New("jenis item harus barang atau jasa")
	}
	if item.PriceRupiah < 0 {
		return errors.New("harga tidak boleh negatif")
	}
	if initialStock < 0 {
		return errors.New("stok awal tidak boleh negatif")
	}
	if !item.TracksStock && initialStock != 0 {
		return errors.New("stok awal hanya berlaku untuk item yang melacak stok")
	}
	if item.AllowsPreorder && !item.TracksStock {
		return errors.New("pre-order hanya berlaku untuk item yang melacak stok")
	}
	if strings.TrimSpace(item.Unit) == "" {
		return errors.New("satuan wajib diisi")
	}
	return nil
}
