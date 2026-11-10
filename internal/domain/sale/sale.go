package sale

import (
	"errors"
	"strings"
	"time"
)

const (
	PaymentCash         = "cash"
	PaymentBankTransfer = "bank_transfer"
	PaymentCard         = "card"
	PaymentEWallet      = "e_wallet"
)

type LineInput struct {
	ItemID   string
	Quantity int64
	PreOrder bool
}

type CheckoutInput struct {
	OperationID        string
	BusinessID         string
	OutletID           string
	Lines              []LineInput
	PaymentMethod      string
	CashReceivedRupiah int64
}

type Line struct {
	ItemID          string
	Name            string
	Quantity        int64
	UnitPriceRupiah int64
	LineTotalRupiah int64
	Fulfillment     string
}

type Receipt struct {
	ID                 string
	OperationID        string
	BusinessID         string
	OutletID           string
	Lines              []Line
	TotalRupiah        int64
	PaymentMethod      string
	CashReceivedRupiah int64
	ChangeRupiah       int64
	CreatedAt          time.Time
}

type JournalEntry struct {
	ID           string
	BusinessID   string
	OutletID     string
	EventType    string
	SourceType   string
	SourceID     string
	AmountRupiah int64
	OccurredAt   time.Time
}

func ValidateCheckout(input CheckoutInput) error {
	if strings.TrimSpace(input.OperationID) == "" {
		return errors.New("ID transaksi wajib diisi")
	}
	if strings.TrimSpace(input.BusinessID) == "" || strings.TrimSpace(input.OutletID) == "" {
		return errors.New("usaha dan outlet wajib dipilih")
	}
	if len(input.Lines) == 0 {
		return errors.New("tambahkan minimal satu item ke transaksi")
	}
	switch input.PaymentMethod {
	case PaymentCash:
		if input.CashReceivedRupiah < 0 {
			return errors.New("jumlah uang diterima tidak boleh negatif")
		}
	case PaymentBankTransfer, PaymentCard, PaymentEWallet:
		if input.CashReceivedRupiah != 0 {
			return errors.New("jumlah tunai hanya boleh diisi untuk pembayaran tunai")
		}
	default:
		return errors.New("metode pembayaran tidak valid")
	}
	seen := make(map[string]struct{}, len(input.Lines))
	for _, line := range input.Lines {
		if strings.TrimSpace(line.ItemID) == "" {
			return errors.New("item transaksi tidak valid")
		}
		if line.Quantity < 1 {
			return errors.New("jumlah item harus minimal satu")
		}
		if _, exists := seen[line.ItemID]; exists {
			return errors.New("item duplikat harus digabung sebelum checkout")
		}
		seen[line.ItemID] = struct{}{}
	}
	return nil
}
