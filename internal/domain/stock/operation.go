package stock

import (
	"errors"
	"strings"
)

const (
	OperationRestock    = "restock"
	OperationAdjustment = "adjustment"
)

type Operation struct {
	OperationID   string
	BusinessID    string
	OutletID      string
	ItemID        string
	Type          string
	QuantityDelta int64
	Reason        string
}

type Result struct {
	OperationID   string
	QuantityAfter int64
}

func Validate(operation Operation) error {
	if strings.TrimSpace(operation.OperationID) == "" || strings.TrimSpace(operation.BusinessID) == "" || strings.TrimSpace(operation.OutletID) == "" || strings.TrimSpace(operation.ItemID) == "" {
		return errors.New("usaha, outlet, item, dan ID operasi wajib diisi")
	}
	switch operation.Type {
	case OperationRestock:
		if operation.QuantityDelta <= 0 {
			return errors.New("jumlah stok masuk harus lebih dari nol")
		}
	case OperationAdjustment:
		if operation.QuantityDelta == 0 {
			return errors.New("penyesuaian stok tidak boleh nol")
		}
	default:
		return errors.New("jenis operasi stok tidak valid")
	}
	if strings.TrimSpace(operation.Reason) == "" {
		return errors.New("alasan perubahan stok wajib diisi")
	}
	return nil
}
