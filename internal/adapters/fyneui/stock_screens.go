package fyneui

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	stockapp "github.com/satubakul/desktop-pos-community/internal/application/stock"
	catalogdomain "github.com/satubakul/desktop-pos-community/internal/domain/catalog"
	stockdomain "github.com/satubakul/desktop-pos-community/internal/domain/stock"
)

func (a *App) showStock(active Business) {
	outlet, ok := firstOutlet(active)
	if !ok {
		a.showError(fmt.Errorf("usaha %s belum memiliki outlet", active.Name))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	items, err := a.catalog.List(ctx, active.ID, outlet.ID)
	cancel()
	if err != nil {
		a.showError(err)
		return
	}
	trackedItems := make([]catalogdomain.Item, 0, len(items))
	for _, item := range items {
		if item.TracksStock {
			trackedItems = append(trackedItems, item)
		}
	}
	if len(trackedItems) == 0 {
		openCatalog := widget.NewButton("Buka katalog", func() { a.showCreateItem(active) })
		openCatalog.Importance = widget.HighImportance
		backHome := widget.NewButton("Kembali ke ringkasan", a.showHome)
		backHome.Importance = widget.LowImportance
		a.setPage(container.NewPadded(container.NewVBox(
			goldText("◆ PERSEDIAAN", 12),
			pageTitle("Stok"),
			wrappedLabel("Belum ada barang yang melacak stok."),
			hairline(),
			openCatalog,
			backHome,
		)), active, "stock")
		return
	}

	operationID, err := stockapp.NewOperationID()
	if err != nil {
		a.showError(err)
		return
	}
	itemLabels := make([]string, 0, len(trackedItems))
	itemsByLabel := make(map[string]catalogdomain.Item, len(trackedItems))
	for _, item := range trackedItems {
		label := fmt.Sprintf("%s · stok %d %s · %s", item.Name, item.StockOnHand, item.Unit, shortID(item.ID))
		itemLabels = append(itemLabels, label)
		itemsByLabel[label] = item
	}
	itemSelect := widget.NewSelect(itemLabels, nil)
	itemSelect.PlaceHolder = "Pilih barang"
	operationType := widget.NewSelect([]string{"Stok masuk", "Penyesuaian"}, nil)
	operationType.SetSelected("Stok masuk")
	direction := widget.NewSelect([]string{"Tambah", "Kurangi"}, nil)
	direction.SetSelected("Tambah")
	direction.Disable()
	operationType.OnChanged = func(selected string) {
		if selected == "Penyesuaian" {
			direction.Enable()
			return
		}
		direction.SetSelected("Tambah")
		direction.Disable()
	}
	quantity := widget.NewEntry()
	quantity.SetText("1")
	reason := widget.NewEntry()
	reason.SetPlaceHolder("Contoh: pembelian barang atau hasil stok opname")
	form := widget.NewForm(
		widget.NewFormItem("Barang", itemSelect),
		widget.NewFormItem("Operasi", operationType),
		widget.NewFormItem("Arah penyesuaian", direction),
		widget.NewFormItem("Jumlah", quantity),
		widget.NewFormItem("Alasan", reason),
	)
	form.SubmitText = "Simpan perubahan stok"
	form.OnSubmit = func() {
		item, exists := itemsByLabel[itemSelect.Selected]
		if !exists {
			a.showError(fmt.Errorf("pilih barang"))
			return
		}
		amount, err := parseRupiah(quantity.Text, false)
		if err != nil || amount < 1 {
			a.showError(fmt.Errorf("jumlah harus berupa bilangan minimal satu"))
			return
		}
		delta := amount
		typeKey := stockdomain.OperationRestock
		if operationType.Selected == "Penyesuaian" {
			typeKey = stockdomain.OperationAdjustment
			if direction.Selected == "Kurangi" {
				delta = -amount
			}
		}
		operation := stockdomain.Operation{
			OperationID:   operationID,
			BusinessID:    active.ID,
			OutletID:      outlet.ID,
			ItemID:        item.ID,
			Type:          typeKey,
			QuantityDelta: delta,
			Reason:        reason.Text,
		}
		operationCtx, operationCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer operationCancel()
		result, err := a.stock.Record(operationCtx, operation)
		if err != nil {
			a.showError(err)
			return
		}
		a.successDialog("Stok tersimpan", "STOK TERSIMPAN",
			fmt.Sprintf("%d %s", result.QuantityAfter, item.Unit),
			[]string{fmt.Sprintf("%s diperbarui.", item.Name)})
		a.showStock(active)
	}
	back := widget.NewButton("Kembali ke ringkasan", a.showHome)
	back.Importance = widget.LowImportance
	content := container.NewVBox(
		goldText("◆ PERSEDIAAN", 12),
		pageTitle("Stok masuk dan penyesuaian"),
		wrappedLabel(fmt.Sprintf("Usaha: %s · Outlet: %s. Semua perubahan stok disimpan sebagai pergerakan dan jurnal.", active.Name, outlet.Name)),
		hairline(),
		form,
		back,
	)
	a.setPage(container.NewPadded(container.NewVScroll(content)), active, "stock")
}
