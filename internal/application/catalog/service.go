package catalog

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	domain "github.com/satubakul/desktop-pos-community/internal/domain/catalog"
)

type Repository interface {
	CreateItem(context.Context, domain.Item, string, int64, *domain.ImageAsset) error
	ListItems(context.Context, string, string) ([]domain.Item, error)
	GetItemImage(context.Context, string, string) (domain.ImageAsset, error)
}

type CreateInput struct {
	BusinessID     string
	OutletID       string
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
	InitialStock   int64
	Image          *domain.ImageAsset
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Item, error) {
	var asset *domain.ImageAsset
	if input.Image != nil {
		copy := *input.Image
		copy.Data = append([]byte(nil), input.Image.Data...)
		if err := validateImageAsset(&copy); err != nil {
			return domain.Item{}, err
		}
		asset = &copy
	}
	id, err := newID()
	if err != nil {
		return domain.Item{}, err
	}
	item := domain.Item{
		ID:             id,
		BusinessID:     strings.TrimSpace(input.BusinessID),
		Name:           strings.TrimSpace(input.Name),
		Description:    strings.TrimSpace(input.Description),
		Category:       strings.TrimSpace(input.Category),
		SKU:            strings.TrimSpace(input.SKU),
		Barcode:        strings.TrimSpace(input.Barcode),
		Unit:           strings.TrimSpace(input.Unit),
		Type:           input.Type,
		PriceRupiah:    input.PriceRupiah,
		TracksStock:    input.TracksStock,
		AllowsPreorder: input.AllowsPreorder,
		Active:         true,
		StockOnHand:    input.InitialStock,
		ImageFilename:  imageFilename(asset),
		CreatedAt:      time.Now().UTC(),
	}
	if item.Unit == "" {
		item.Unit = "pcs"
	}
	if err := domain.Validate(item, input.InitialStock); err != nil {
		return domain.Item{}, err
	}
	if strings.TrimSpace(input.OutletID) == "" {
		return domain.Item{}, errors.New("outlet wajib dipilih")
	}
	if err := s.repository.CreateItem(ctx, item, input.OutletID, input.InitialStock, asset); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func imageFilename(asset *domain.ImageAsset) string {
	if asset == nil {
		return ""
	}
	return asset.FileName
}

func validateImageAsset(asset *domain.ImageAsset) error {
	asset.FileName = filepath.Base(strings.TrimSpace(asset.FileName))
	if asset.FileName == "" || asset.FileName == "." {
		return errors.New("nama file gambar tidak valid")
	}
	if len(asset.Data) == 0 || len(asset.Data) > domain.MaxImageSize {
		return fmt.Errorf("gambar harus berukuran maksimal %d MB", domain.MaxImageSize/(1<<20))
	}
	contentType := http.DetectContentType(asset.Data)
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/gif" {
		return errors.New("format gambar harus JPG, PNG, atau GIF")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(asset.Data))
	if err != nil {
		return errors.New("file gambar tidak dapat dibaca")
	}
	if config.Width < 1 || config.Height < 1 || config.Width > 30000 || config.Height > 30000 || int64(config.Width) > 50000000/int64(config.Height) {
		return errors.New("dimensi gambar terlalu besar")
	}
	if format == "jpeg" {
		asset.ContentType = "image/jpeg"
	} else if format == "png" {
		asset.ContentType = "image/png"
	} else if format == "gif" {
		asset.ContentType = "image/gif"
	} else {
		return errors.New("format gambar harus JPG, PNG, atau GIF")
	}
	if asset.ContentType != contentType {
		return errors.New("isi file tidak sesuai dengan format gambar")
	}
	return nil
}

func (s *Service) List(ctx context.Context, businessID, outletID string) ([]domain.Item, error) {
	if strings.TrimSpace(businessID) == "" || strings.TrimSpace(outletID) == "" {
		return nil, errors.New("usaha dan outlet wajib dipilih")
	}
	return s.repository.ListItems(ctx, businessID, outletID)
}

func (s *Service) Image(ctx context.Context, businessID, itemID string) (domain.ImageAsset, error) {
	if strings.TrimSpace(businessID) == "" || strings.TrimSpace(itemID) == "" {
		return domain.ImageAsset{}, errors.New("usaha dan item wajib dipilih")
	}
	return s.repository.GetItemImage(ctx, businessID, itemID)
}

func newID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}
