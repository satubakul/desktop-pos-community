package stock

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	domain "github.com/satubakul/desktop-pos-community/internal/domain/stock"
)

type Repository interface {
	Record(context.Context, domain.Operation, string) (domain.Result, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func NewOperationID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func (s *Service) Record(ctx context.Context, operation domain.Operation) (domain.Result, error) {
	operation.OperationID = strings.TrimSpace(operation.OperationID)
	operation.BusinessID = strings.TrimSpace(operation.BusinessID)
	operation.OutletID = strings.TrimSpace(operation.OutletID)
	operation.ItemID = strings.TrimSpace(operation.ItemID)
	operation.Reason = strings.TrimSpace(operation.Reason)
	if err := domain.Validate(operation); err != nil {
		return domain.Result{}, err
	}
	canonical, err := json.Marshal(operation)
	if err != nil {
		return domain.Result{}, errors.New("gagal membentuk data operasi stok")
	}
	hash := sha256.Sum256(canonical)
	return s.repository.Record(ctx, operation, hex.EncodeToString(hash[:]))
}
