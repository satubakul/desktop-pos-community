package fyneui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	application "github.com/satubakul/desktop-pos-community/internal/application/business"
	catalogapp "github.com/satubakul/desktop-pos-community/internal/application/catalog"
	domain "github.com/satubakul/desktop-pos-community/internal/domain/business"
	catalogdomain "github.com/satubakul/desktop-pos-community/internal/domain/catalog"
	saledomain "github.com/satubakul/desktop-pos-community/internal/domain/sale"
	stockdomain "github.com/satubakul/desktop-pos-community/internal/domain/stock"
)

type stubBusinessService struct {
	active domain.Business
}

func (s stubBusinessService) Create(context.Context, application.CreateInput) (domain.Business, error) {
	return domain.Business{}, nil
}

func (s stubBusinessService) List(context.Context) ([]domain.Business, error) {
	return []domain.Business{s.active}, nil
}

func (s stubBusinessService) ActiveID(context.Context) (string, error) {
	return s.active.ID, nil
}

func (s stubBusinessService) SelectActive(context.Context, string) error { return nil }

type stubCatalogService struct {
	items []catalogdomain.Item
	image catalogdomain.ImageAsset
}

func (s stubCatalogService) Create(context.Context, catalogapp.CreateInput) (catalogdomain.Item, error) {
	return catalogdomain.Item{}, nil
}

func (s stubCatalogService) List(context.Context, string, string) ([]catalogdomain.Item, error) {
	return s.items, nil
}

func (s stubCatalogService) Image(context.Context, string, string) (catalogdomain.ImageAsset, error) {
	return s.image, nil
}

type stubSalesService struct {
	journal []saledomain.JournalEntry
}

func (stubSalesService) Checkout(context.Context, saledomain.CheckoutInput) (saledomain.Receipt, error) {
	return saledomain.Receipt{}, nil
}

func (s stubSalesService) ListJournal(context.Context, string) ([]saledomain.JournalEntry, error) {
	return s.journal, nil
}

type stubStockService struct{}

func (stubStockService) Record(context.Context, stockdomain.Operation) (stockdomain.Result, error) {
	return stockdomain.Result{}, nil
}

// walkObjects visits every object in the tree. Plain containers expose
// Objects directly; every other layout (Border, Split, Scroll, Card, …)
// is a widget whose laid-out children come from its renderer.
func walkObjects(root fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	visit(root)
	if c, ok := root.(*fyne.Container); ok {
		for _, child := range c.Objects {
			walkObjects(child, visit)
		}
		return
	}
	if w, ok := root.(fyne.Widget); ok {
		for _, child := range test.WidgetRenderer(w).Objects() {
			walkObjects(child, visit)
		}
	}
}

func findButtons(root fyne.CanvasObject, label string) []*widget.Button {
	var out []*widget.Button
	walkObjects(root, func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && b.Text == label {
			out = append(out, b)
		}
	})
	return out
}

// findCardGrids returns containers whose children are all product cards.
func findCardGrids(root fyne.CanvasObject) []*fyne.Container {
	var out []*fyne.Container
	walkObjects(root, func(o fyne.CanvasObject) {
		c, ok := o.(*fyne.Container)
		if !ok || len(c.Objects) == 0 {
			return
		}
		for _, child := range c.Objects {
			if _, ok := child.(*widget.Card); !ok {
				return
			}
		}
		out = append(out, c)
	})
	return out
}

func newTestUI(t *testing.T, items []catalogdomain.Item, journal []saledomain.JournalEntry, image catalogdomain.ImageAsset) (*App, Business) {
	t.Helper()
	a := test.NewApp()
	t.Cleanup(a.Quit)
	a.Settings().SetTheme(posTheme{})

	active := domain.Business{
		ID:   "business-1",
		Name: "Toko Uji",
		Outlets: []domain.Outlet{
			{ID: "outlet-1", BusinessID: "business-1", Name: "Outlet Uji"},
		},
	}
	ui := &App{
		businesses: stubBusinessService{active: active},
		catalog:    stubCatalogService{items: items, image: image},
		sales:      stubSalesService{journal: journal},
		stock:      stubStockService{},
		app:        a,
	}
	return ui, active
}

func openCheckout(t *testing.T, items []catalogdomain.Item) fyne.Window {
	t.Helper()
	ui, active := newTestUI(t, items, nil, catalogdomain.ImageAsset{})
	w := ui.app.NewWindow("Kasir")
	ui.window = w
	ui.showCheckout(active)
	// Tall window: the scroll viewport exceeds the grid's natural height,
	// which is exactly when the old layout stretched the cards.
	w.Resize(fyne.NewSize(1360, 1100))
	return w
}

func openJournal(t *testing.T, journal []saledomain.JournalEntry) fyne.Window {
	t.Helper()
	ui, active := newTestUI(t, nil, journal, catalogdomain.ImageAsset{})
	w := ui.app.NewWindow("Jurnal")
	ui.window = w
	ui.showJournal(active)
	w.Resize(fyne.NewSize(1360, 900))
	return w
}

func checkoutTestItems() []catalogdomain.Item {
	return []catalogdomain.Item{
		{ID: "item-1", BusinessID: "business-1", Name: "Komputer", Unit: "pcs", Type: catalogdomain.TypeGood, PriceRupiah: 2000000, TracksStock: true, StockOnHand: 98},
		{ID: "item-2", BusinessID: "business-1", Name: "Mouse", Unit: "pcs", Type: catalogdomain.TypeGood, PriceRupiah: 150000, TracksStock: true, StockOnHand: 50},
	}
}

// TestCashierLayoutRegression guards the two cashier layout defects reported
// against the real checkout screen: the product grid must keep its natural
// height instead of stretching to the scroll viewport, and the cart stepper
// buttons (−/+) must keep button height instead of filling the whole row.
func TestCashierLayoutRegression(t *testing.T) {
	items := checkoutTestItems()
	w := openCheckout(t, items)

	grids := findCardGrids(w.Canvas().Content())
	if len(grids) == 0 {
		t.Fatal("product grid not found")
	}
	for _, g := range grids {
		if got := g.Size().Height - g.MinSize().Height; got > 1 {
			t.Errorf("product grid stretched: size=%v min=%v", g.Size(), g.MinSize())
		}
	}

	addButtons := findButtons(w.Canvas().Content(), "Tambah")
	if len(addButtons) != len(items) {
		t.Fatalf("got %d Tambah buttons, want %d", len(addButtons), len(items))
	}
	test.Tap(addButtons[0])
	test.Tap(addButtons[0])

	var minus *widget.Button
	for _, b := range findButtons(w.Canvas().Content(), "−") {
		minus = b
		break
	}
	if minus == nil {
		t.Fatal("cart stepper button not found after adding to cart")
	}
	if got := minus.Size().Height - minus.MinSize().Height; got > 1 {
		t.Errorf("cart stepper stretched: size=%v min=%v", minus.Size(), minus.MinSize())
	}
}

// TestCheckoutSuccessDialog guards the reported tiny success popup: after a
// cash checkout the receipt dialog must be roomy, not squeezed to its text.
func TestCheckoutSuccessDialog(t *testing.T) {
	w := openCheckout(t, checkoutTestItems()[:1])

	addButtons := findButtons(w.Canvas().Content(), "Tambah")
	if len(addButtons) != 1 {
		t.Fatalf("got %d Tambah buttons, want 1", len(addButtons))
	}
	test.Tap(addButtons[0])

	var cash *widget.Entry
	walkObjects(w.Canvas().Content(), func(o fyne.CanvasObject) {
		if e, ok := o.(*widget.Entry); ok && e.PlaceHolder == "Contoh: 50.000" && cash == nil {
			cash = e
		}
	})
	if cash == nil {
		t.Fatal("cash entry not found")
	}
	test.Type(cash, "5000000")

	payButtons := findButtons(w.Canvas().Content(), "Bayar · Rp 2.000.000")
	if len(payButtons) != 1 {
		t.Fatal("pay button not enabled after cash input")
	}
	test.Tap(payButtons[0])

	top := w.Canvas().Overlays().Top()
	if top == nil {
		t.Fatal("no dialog overlay after checkout")
	}
	if got := top.MinSize().Width; got < 380 {
		t.Errorf("success dialog squeezed: overlay min width=%v, want >= 380", got)
	}
}

func gridColumns(w fyne.Window) int {
	xs := map[float32]struct{}{}
	walkObjects(w.Canvas().Content(), func(o fyne.CanvasObject) {
		if _, ok := o.(*widget.Card); ok {
			xs[o.Position().X] = struct{}{}
		}
	})
	return len(xs)
}

// TestProductGridResponsiveColumns guards the configurable grid: at most
// maxProductColumns while they fit, fewer when the window narrows.
func TestProductGridResponsiveColumns(t *testing.T) {
	names := []string{"Komputer", "Mouse", "Keyboard", "Monitor", "Printer", "Kabel"}
	prices := []int64{2000000, 150000, 350000, 1800000, 1200000, 25000}
	items := make([]catalogdomain.Item, 0, len(names))
	for i := range names {
		items = append(items, catalogdomain.Item{
			ID: fmt.Sprintf("item-%d", i+1), BusinessID: "business-1",
			Name: names[i], Unit: "pcs", Type: catalogdomain.TypeGood,
			PriceRupiah: prices[i], TracksStock: true, StockOnHand: 50,
		})
	}
	w := openCheckout(t, items)
	if got := gridColumns(w); got != 3 {
		t.Errorf("1360px window: got %d columns, want 3", got)
	}
	w.Resize(fyne.NewSize(2000, 1100))
	if got := gridColumns(w); got != maxProductColumns {
		t.Errorf("2000px window: got %d columns, want %d", got, maxProductColumns)
	}
	w.Resize(fyne.NewSize(800, 1100))
	if got := gridColumns(w); got != 1 {
		t.Errorf("800px window: got %d columns, want 1", got)
	}
}

// TestJournalLedger guards the premium ledger rows: event name, amount and
// reference render per entry, each with its event-color badge.
func TestJournalLedger(t *testing.T) {
	at := time.Date(2026, 11, 10, 21, 33, 0, 0, time.UTC)
	entries := []saledomain.JournalEntry{
		{EventType: "sale", SourceID: "saleop01", AmountRupiah: 4000000, OccurredAt: at},
		{EventType: "payment", SourceID: "payop002", AmountRupiah: 4000000, OccurredAt: at},
		{EventType: "stock_restock", SourceID: "stock003", AmountRupiah: 0, OccurredAt: at},
	}
	w := openJournal(t, entries)

	texts := map[string]bool{}
	walkObjects(w.Canvas().Content(), func(o fyne.CanvasObject) {
		switch o := o.(type) {
		case *widget.Label:
			texts[o.Text] = true
		case *canvas.Text:
			texts[o.Text] = true
		}
	})
	for _, want := range []string{"Penjualan", "Pembayaran", "Stok masuk", "Rp 4.000.000", "#saleop", "#payop0", "#stock0"} {
		if !texts[want] {
			t.Errorf("journal text %q not rendered", want)
		}
	}

	badges := map[color.NRGBA]bool{}
	walkObjects(w.Canvas().Content(), func(o fyne.CanvasObject) {
		r, ok := o.(*canvas.Rectangle)
		if !ok || r.Size().Height < 40 {
			return
		}
		if c, ok := r.FillColor.(color.NRGBA); ok {
			badges[c] = true
		}
	})
	for _, want := range []color.NRGBA{posPrimaryColor, posGoldColor, {R: 150, G: 158, B: 153, A: 255}} {
		if !badges[want] {
			t.Errorf("journal badge color %v not rendered", want)
		}
	}
}

// TestJournalEmpty guards the empty journal card.
func TestJournalEmpty(t *testing.T) {
	w := openJournal(t, nil)
	found := false
	walkObjects(w.Canvas().Content(), func(o fyne.CanvasObject) {
		// Card titles render through the card renderer as canvas text.
		if t, ok := o.(*canvas.Text); ok && t.Text == "Jurnal masih kosong" {
			found = true
		}
	})
	if !found {
		t.Error("empty journal card not rendered")
	}
}

func openCatalog(t *testing.T, items []catalogdomain.Item) fyne.Window {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	ui, active := newTestUI(t, items, nil, catalogdomain.ImageAsset{FileName: "foto.png", Data: buf.Bytes()})
	w := ui.app.NewWindow("Produk")
	ui.window = w
	ui.showCatalog(active)
	w.Resize(fyne.NewSize(1360, 900))
	return w
}

// TestCatalogRows guards the premium product rows: name, category, price
// and stock render per item, with a photo thumbnail only for the item
// that has one.
func TestCatalogRows(t *testing.T) {
	items := []catalogdomain.Item{
		{ID: "item-1", BusinessID: "business-1", Name: "Komputer", Category: "Elektronik", Unit: "pcs", Type: catalogdomain.TypeGood, PriceRupiah: 2000000, TracksStock: true, StockOnHand: 97},
		{ID: "item-2", BusinessID: "business-1", Name: "Servis", Unit: "unit", Type: catalogdomain.TypeService, PriceRupiah: 150000},
	}
	// Only the first item carries a photo.
	items[0].ImageFilename = "komputer.png"
	w := openCatalog(t, items)

	texts := map[string]bool{}
	walkObjects(w.Canvas().Content(), func(o fyne.CanvasObject) {
		switch o := o.(type) {
		case *widget.Label:
			texts[o.Text] = true
		case *canvas.Text:
			texts[o.Text] = true
		}
	})
	for _, want := range []string{"Komputer", "Servis", "Barang · Elektronik", "Jasa", "Rp 2.000.000", "Rp 150.000", "Tersisa 97 pcs", "Jasa · tanpa stok"} {
		if !texts[want] {
			t.Errorf("catalog text %q not rendered", want)
		}
	}

	photos := 0
	walkObjects(w.Canvas().Content(), func(o fyne.CanvasObject) {
		// Only product thumbnails (60px min) count, not widget icons.
		if img, ok := o.(*canvas.Image); ok && img.MinSize().Width >= 59 {
			if img.Visible() && img.Resource != nil {
				photos++
			}
		}
	})
	if photos != 1 {
		t.Errorf("got %d visible product photos, want 1", photos)
	}
}

// TestCatalogEmpty guards the empty catalog card.
func TestCatalogEmpty(t *testing.T) {
	w := openCatalog(t, nil)
	found := false
	walkObjects(w.Canvas().Content(), func(o fyne.CanvasObject) {
		if t, ok := o.(*canvas.Text); ok && t.Text == "Katalog masih kosong" {
			found = true
		}
	})
	if !found {
		t.Error("empty catalog card not rendered")
	}
}
