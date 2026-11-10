package sales

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	domain "github.com/satubakul/desktop-pos-community/internal/domain/sale"
)

type Repository interface {
	Checkout(context.Context, domain.CheckoutInput, string) (domain.Receipt, error)
	ListJournal(context.Context, string) ([]domain.JournalEntry, error)
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

func (s *Service) Checkout(ctx context.Context, input domain.CheckoutInput) (domain.Receipt, error) {
	if err := domain.ValidateCheckout(input); err != nil {
		return domain.Receipt{}, err
	}
	return s.repository.Checkout(ctx, input, requestHash(input))
}

func (s *Service) ListJournal(ctx context.Context, businessID string) ([]domain.JournalEntry, error) {
	if strings.TrimSpace(businessID) == "" {
		return nil, errors.New("usaha wajib dipilih")
	}
	return s.repository.ListJournal(ctx, businessID)
}

func requestHash(input domain.CheckoutInput) string {
	var canonical strings.Builder
	fmt.Fprintf(&canonical, "%s\n%s\n%s\n%d\n", input.BusinessID, input.OutletID, input.PaymentMethod, input.CashReceivedRupiah)
	for _, line := range input.Lines {
		fmt.Fprintf(&canonical, "%s:%d:%t\n", line.ItemID, line.Quantity, line.PreOrder)
	}
	hash := sha256.Sum256([]byte(canonical.String()))
	return hex.EncodeToString(hash[:])
}
