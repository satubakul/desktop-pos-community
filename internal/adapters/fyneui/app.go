package fyneui

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	application "github.com/satubakul/desktop-pos-community/internal/application/business"
	catalogapp "github.com/satubakul/desktop-pos-community/internal/application/catalog"
	domain "github.com/satubakul/desktop-pos-community/internal/domain/business"
	catalogdomain "github.com/satubakul/desktop-pos-community/internal/domain/catalog"
	saledomain "github.com/satubakul/desktop-pos-community/internal/domain/sale"
	stockdomain "github.com/satubakul/desktop-pos-community/internal/domain/stock"
)

type BusinessService interface {
	Create(context.Context, application.CreateInput) (domain.Business, error)
	List(context.Context) ([]domain.Business, error)
	ActiveID(context.Context) (string, error)
	SelectActive(context.Context, string) error
}

type CatalogService interface {
	Create(context.Context, catalogapp.CreateInput) (catalogdomain.Item, error)
	List(context.Context, string, string) ([]catalogdomain.Item, error)
	Image(context.Context, string, string) (catalogdomain.ImageAsset, error)
}

type SalesService interface {
	Checkout(context.Context, saledomain.CheckoutInput) (saledomain.Receipt, error)
	ListJournal(context.Context, string) ([]saledomain.JournalEntry, error)
}

type StockService interface {
	Record(context.Context, stockdomain.Operation) (stockdomain.Result, error)
}

type Business = domain.Business
type Profile = domain.Profile
type Outlet = domain.Outlet

type App struct {
	businesses BusinessService
	catalog    CatalogService
	sales      SalesService
	stock      StockService
	app        fyne.App
	window     fyne.Window
}

func New(businesses BusinessService, catalog CatalogService, sales SalesService, stock StockService) *App {
	return &App{businesses: businesses, catalog: catalog, sales: sales, stock: stock}
}

// Focus brings the main window forward. It is safe to call any time and
// from any goroutine; before Run it is a no-op. Note the driver ignores
// it on Wayland, where the compositor owns window focus.
func (a *App) Focus() {
	if a.window == nil {
		return
	}
	a.window.RequestFocus()
}

func (a *App) Run() {
	a.app = fyneapp.New()
	a.app.Settings().SetTheme(posTheme{})
	a.window = a.app.NewWindow("Kasir Kita — Premium POS")
	a.window.Resize(fyne.NewSize(1360, 860))
	a.window.CenterOnScreen()
	a.window.SetMaster()
	a.window.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu("Aplikasi",
			fyne.NewMenuItem("Tutup Aplikasi", func() { a.app.Quit() }),
		),
	))
	a.showStartup()
	a.window.ShowAndRun()
}

func (a *App) setPage(content fyne.CanvasObject, active Business, current string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	businesses, err := a.businesses.List(ctx)
	cancel()
	if err != nil {
		a.showError(err)
		return
	}
	labels := make([]string, 0, len(businesses))
	idsByLabel := make(map[string]string, len(businesses))
	activeLabel := ""
	for _, business := range businesses {
		label := workspaceLabel(business)
		labels = append(labels, label)
		idsByLabel[label] = business.ID
		if business.ID == active.ID {
			activeLabel = label
		}
	}
	switcher := widget.NewSelect(labels, func(label string) {
		id := idsByLabel[label]
		if id == "" || id == active.ID {
			return
		}
		switchCtx, switchCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer switchCancel()
		if err := a.businesses.SelectActive(switchCtx, id); err != nil {
			a.showError(err)
			return
		}
		a.showHome()
	})
	if activeLabel != "" {
		switcher.SetSelected(activeLabel)
	}

	navItem := func(label string, icon fyne.Resource, key string, open func()) fyne.CanvasObject {
		button := widget.NewButtonWithIcon(label, icon, open)
		button.Alignment = widget.ButtonAlignLeading
		button.Importance = widget.LowImportance
		indicator := canvas.NewRectangle(posSurfaceColor)
		indicator.SetMinSize(fyne.NewSize(3, 34))
		if current == key {
			button.Importance = widget.HighImportance
			indicator = canvas.NewRectangle(posGoldColor)
			indicator.SetMinSize(fyne.NewSize(3, 34))
		}
		return container.NewBorder(nil, nil, indicator, nil, button)
	}
	nav := container.NewVBox(
		captionLabel("MENU"),
		navItem("Ringkasan", theme.HomeIcon(), "home", a.showHome),
		navItem("Kasir", theme.DocumentCreateIcon(), "cashier", func() { a.showCashier(active) }),
		navItem("Produk", theme.GridIcon(), "products", func() { a.showCatalog(active) }),
		navItem("Stok", theme.StorageIcon(), "stock", func() { a.showStock(active) }),
		navItem("Jurnal", theme.HistoryIcon(), "journal", func() { a.showJournal(active) }),
	)

	// Dark brand plate: the single luxury accent in an otherwise clean sidebar.
	brandPlate := container.NewStack(
		canvas.NewRectangle(posBrandDark),
		container.NewPadded(container.NewVBox(
			goldText("◆ KASIR KITA", 13),
			brandTitle(active.Name),
			brandCaption("PREMIUM POINT OF SALE"),
		)),
	)

	width := canvas.NewRectangle(posSurfaceColor)
	width.SetMinSize(fyne.NewSize(248, 1))
	sidebarBody := container.NewVBox(
		width,
		captionLabel("USAHA AKTIF"),
		switcher,
		hairline(),
		nav,
	)
	sidebarContent := container.NewBorder(
		container.NewVBox(brandPlate, goldRule()),
		container.NewVBox(
			hairline(),
			brandCaption("MODE LOKAL · DATA DI PERANGKAT INI"),
			widget.NewButtonWithIcon("Tutup Aplikasi", theme.LogoutIcon(), func() { a.app.Quit() }),
		),
		nil,
		nil,
		sidebarBody,
	)
	sidebar := container.NewStack(
		canvas.NewRectangle(posSurfaceColor),
		container.NewPadded(sidebarContent),
	)
	// Hairline separating the sidebar from the content area.
	shell := container.NewBorder(nil, nil, sidebar, nil, container.NewPadded(content))
	a.window.SetContent(shell)
}
