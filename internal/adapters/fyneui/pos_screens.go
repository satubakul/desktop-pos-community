package fyneui

import (
	"context"
	"fmt"
	"image/color"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	catalogapp "github.com/satubakul/desktop-pos-community/internal/application/catalog"
	salesapp "github.com/satubakul/desktop-pos-community/internal/application/sales"
	catalogdomain "github.com/satubakul/desktop-pos-community/internal/domain/catalog"
	saledomain "github.com/satubakul/desktop-pos-community/internal/domain/sale"
)

type cartLine struct {
	item     catalogdomain.Item
	quantity int64
	preOrder bool
}

// Pengaturan grid produk di layar kasir.
const (
	// maxProductColumns adalah jumlah kolom terbanyak: grid menampilkan
	// sebanyak ini selama muat, berkurang otomatis saat jendela disempitkan.
	maxProductColumns = 5
	// minProductCardWidth adalah lebar minimum satu kartu (px) yang
	// menentukan kapan kolom dikurangi.
	minProductCardWidth = 190
)

// responsiveGrid membungkus grid produk dan menyesuaikan jumlah kolom
// dengan lebar yang tersedia setiap kali layout berjalan.
type responsiveGrid struct {
	grid *fyne.Container
	cols int
}

func (r *responsiveGrid) MinSize(objects []fyne.CanvasObject) fyne.Size {
	// Lebar selalu basis satu kolom agar jendela tetap bisa disempitkan
	// (min multi-kolom mengunci jendela tetap lebar sehingga kolom tak
	// pernah berkurang); tinggi mengikuti kolom aktif agar tak ada
	// ruang mati maupun scrollbar hantu. Kelebihan tinggi saat sempit
	// ditampung scroll vertikal.
	narrow := layout.NewGridLayoutWithColumns(1).MinSize(r.grid.Objects)
	cols := r.cols
	if cols < 1 {
		cols = 1
	}
	current := layout.NewGridLayoutWithColumns(cols).MinSize(r.grid.Objects)
	return fyne.NewSize(narrow.Width, current.Height)
}

func (r *responsiveGrid) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	padding := theme.Padding()
	cols := int((size.Width + padding) / (minProductCardWidth + padding))
	if cols < 1 {
		cols = 1
	}
	if cols > maxProductColumns {
		cols = maxProductColumns
	}
	if cols != r.cols {
		r.cols = cols
		r.grid.Layout = layout.NewGridLayoutWithColumns(cols)
	}
	for _, o := range objects {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(fyne.NewSize(size.Width, o.MinSize().Height))
	}
}

func (a *App) showCatalog(active Business) {
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

	title := container.NewVBox(
		goldText("◆ KATALOG", 12),
		pageTitle("Barang dan jasa"),
		wrappedLabel(fmt.Sprintf("%s · Outlet %s · %d item", active.Name, outlet.Name, len(items))),
		hairline(),
	)
	add := widget.NewButton("Tambah barang atau jasa", func() { a.showCreateItem(active) })
	add.Importance = widget.HighImportance
	openCashier := widget.NewButton("Buka Kasir", func() { a.showCashier(active) })
	back := widget.NewButton("Kembali ke ringkasan", a.showHome)
	back.Importance = widget.LowImportance
	footer := container.NewHBox(add, openCashier, back)
	if len(items) == 0 {
		a.setPage(container.NewCenter(widget.NewCard(
			"Katalog masih kosong",
			"Tambahkan produk pertama untuk mulai berjualan",
			container.NewVBox(add, openCashier, back),
		)), active, "products")
		return
	}
	imageResources := make(map[string]fyne.Resource)
	imageCtx, imageCancel := context.WithTimeout(context.Background(), 5*time.Second)
	for _, item := range items {
		if item.ImageFilename == "" {
			continue
		}
		asset, imageErr := a.catalog.Image(imageCtx, active.ID, item.ID)
		if imageErr == nil {
			imageResources[item.ID] = fyne.NewStaticResource(asset.FileName, asset.Data)
		}
	}
	imageCancel()
	rows := make(map[fyne.CanvasObject]*productRowView)
	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject {
			img := &canvas.Image{FillMode: canvas.ImageFillContain}
			img.SetMinSize(fyne.NewSize(60, 60))
			img.Hide()
			blank := canvas.NewRectangle(color.Transparent)
			blank.SetMinSize(fyne.NewSize(60, 60))
			view := &productRowView{
				image: img,
				blank: blank,
				name:  widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				sub:   captionLabel(""),
				price: primaryText("", 15),
				stock: captionLabel(""),
			}
			view.price.Alignment = fyne.TextAlignTrailing
			view.stock.Alignment = fyne.TextAlignTrailing
			row := container.NewHBox(
				container.NewStack(blank, img),
				container.NewVBox(view.name, view.sub),
				layout.NewSpacer(),
				container.NewVBox(view.price, view.stock),
			)
			padded := container.NewPadded(row)
			rows[padded] = view
			return padded
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			item := items[id]
			view, ok := rows[object]
			if !ok {
				return
			}
			view.name.SetText(item.Name)
			kind := "Barang"
			if item.Type == catalogdomain.TypeService {
				kind = "Jasa"
			}
			sub := kind
			if category := strings.TrimSpace(item.Category); category != "" {
				sub += " · " + category
			}
			view.sub.Text = sub
			view.sub.Refresh()
			view.price.Text = formatRupiah(item.PriceRupiah)
			view.price.Refresh()
			stock := "Stok tidak dilacak"
			if item.Type == catalogdomain.TypeService {
				stock = "Jasa · tanpa stok"
			} else if item.TracksStock {
				if item.StockOnHand == 0 {
					stock = "Stok habis"
				} else {
					stock = fmt.Sprintf("Tersisa %d %s", item.StockOnHand, item.Unit)
				}
			}
			if item.AllowsPreorder {
				stock += " · Pre-order"
			}
			view.stock.Text = stock
			view.stock.Refresh()
			if resource, ok := imageResources[item.ID]; ok {
				view.image.Resource = resource
				view.image.Show()
				view.blank.Hide()
			} else {
				view.image.Hide()
				view.blank.Show()
			}
			view.image.Refresh()
			view.blank.Refresh()
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		a.showItemImage(active.ID, items[id])
	}
	a.setPage(container.NewBorder(
		title,
		footer,
		nil,
		nil,
		list,
	), active, "products")
}

func (a *App) showItemImage(businessID string, item catalogdomain.Item) {
	if item.ImageFilename == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	asset, err := a.catalog.Image(ctx, businessID, item.ID)
	cancel()
	if err != nil {
		a.showError(err)
		return
	}
	preview := canvas.NewImageFromResource(fyne.NewStaticResource(asset.FileName, asset.Data))
	preview.FillMode = canvas.ImageFillContain
	dialog.ShowCustom("Gambar item: "+item.Name, "Tutup", container.NewGridWrap(fyne.NewSize(420, 320), preview), a.window)
}

func (a *App) showCreateItem(active Business) {
	outlet, ok := firstOutlet(active)
	if !ok {
		a.showError(fmt.Errorf("usaha %s belum memiliki outlet", active.Name))
		return
	}
	name := widget.NewEntry()
	name.SetPlaceHolder("Nama barang atau jasa")
	category := widget.NewEntry()
	category.SetPlaceHolder("Kategori, jika ada")
	sku := widget.NewEntry()
	sku.SetPlaceHolder("SKU, jika ada")
	barcode := widget.NewEntry()
	barcode.SetPlaceHolder("Barcode, jika ada")
	unit := widget.NewEntry()
	unit.SetText("pcs")
	price := widget.NewEntry()
	price.SetPlaceHolder("Contoh: 25000")
	initialStock := widget.NewEntry()
	initialStock.SetText("0")
	tracksStock := widget.NewCheck("Lacak stok untuk item ini", nil)
	allowsPreorder := widget.NewCheck("Izinkan pre-order tanpa cek stok tersedia", nil)
	imageStatus := widget.NewLabel("Belum ada gambar")
	var imageAsset *catalogdomain.ImageAsset
	chooseImage := widget.NewButton("Pilih gambar", func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				a.showError(err)
				return
			}
			if reader == nil {
				return
			}
			defer reader.Close()
			data, err := io.ReadAll(io.LimitReader(reader, int64(catalogdomain.MaxImageSize)+1))
			if err != nil {
				a.showError(err)
				return
			}
			if len(data) > catalogdomain.MaxImageSize {
				a.showError(fmt.Errorf("gambar maksimal %d MB", catalogdomain.MaxImageSize/(1<<20)))
				return
			}
			imageAsset = &catalogdomain.ImageAsset{FileName: reader.URI().Name(), Data: data}
			imageStatus.SetText("Gambar dipilih: " + imageAsset.FileName)
		}, a.window)
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".jpg", ".jpeg", ".png", ".gif"}))
		fileDialog.Show()
	})
	removeImage := widget.NewButton("Hapus pilihan", func() {
		imageAsset = nil
		imageStatus.SetText("Belum ada gambar")
	})
	initialStock.Disable()
	allowsPreorder.Disable()
	tracksStock.OnChanged = func(enabled bool) {
		if enabled {
			initialStock.Enable()
			allowsPreorder.Enable()
			return
		}
		initialStock.Disable()
		allowsPreorder.Disable()
		allowsPreorder.SetChecked(false)
		initialStock.SetText("0")
	}
	itemType := widget.NewSelect([]string{"Barang", "Jasa"}, nil)
	itemType.SetSelected("Barang")

	form := widget.NewForm(
		widget.NewFormItem("Nama *", name),
		widget.NewFormItem("Jenis *", itemType),
		widget.NewFormItem("Kategori", category),
		widget.NewFormItem("SKU", sku),
		widget.NewFormItem("Barcode", barcode),
		widget.NewFormItem("Satuan", unit),
		widget.NewFormItem("Harga jual (Rp) *", price),
		widget.NewFormItem("Stok", tracksStock),
		widget.NewFormItem("Stok awal", initialStock),
		widget.NewFormItem("Pre-order", allowsPreorder),
		widget.NewFormItem("Gambar item", container.NewHBox(imageStatus, chooseImage, removeImage)),
	)
	form.SubmitText = "Simpan item"
	form.OnSubmit = func() {
		priceRupiah, err := parseRupiah(price.Text, false)
		if err != nil {
			a.showError(err)
			return
		}
		stock := int64(0)
		if tracksStock.Checked {
			stock, err = parseRupiah(initialStock.Text, true)
			if err != nil {
				a.showError(err)
				return
			}
		}
		typeKey := catalogdomain.TypeGood
		if itemType.Selected == "Jasa" {
			typeKey = catalogdomain.TypeService
		}
		input := catalogapp.CreateInput{
			BusinessID:     active.ID,
			OutletID:       outlet.ID,
			Name:           name.Text,
			Category:       category.Text,
			SKU:            sku.Text,
			Barcode:        barcode.Text,
			Unit:           unit.Text,
			Type:           typeKey,
			PriceRupiah:    priceRupiah,
			TracksStock:    tracksStock.Checked,
			AllowsPreorder: allowsPreorder.Checked,
			InitialStock:   stock,
			Image:          imageAsset,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := a.catalog.Create(ctx, input); err != nil {
			a.showError(err)
			return
		}
		a.showCatalog(active)
	}
	back := widget.NewButton("Kembali ke katalog", func() { a.showCatalog(active) })
	back.Importance = widget.LowImportance
	form.OnCancel = func() { back.OnTapped() }
	content := container.NewVBox(
		goldText("◆ ITEM BARU", 12),
		pageTitle("Tambah item"),
		wrappedLabel("Stok awal dicatat sebagai pergerakan pembukaan. Untuk pre-order, stok tidak diperiksa dan tidak berubah sampai barang diterima."),
		hairline(),
		form,
		back,
	)
	a.setPage(container.NewPadded(container.NewVScroll(content)), active, "products")
}

func (a *App) showCheckout(active Business) {
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
	imageResources := make(map[string]fyne.Resource)
	imageCtx, imageCancel := context.WithTimeout(context.Background(), 5*time.Second)
	for _, item := range items {
		if item.ImageFilename == "" {
			continue
		}
		asset, imageErr := a.catalog.Image(imageCtx, active.ID, item.ID)
		if imageErr == nil {
			imageResources[item.ID] = fyne.NewStaticResource(asset.FileName, asset.Data)
		}
	}
	imageCancel()
	if len(items) == 0 {
		addFirst := widget.NewButton("Tambah barang atau jasa", func() { a.showCreateItem(active) })
		addFirst.Importance = widget.HighImportance
		backEmpty := widget.NewButton("Kembali ke ringkasan", a.showHome)
		backEmpty.Importance = widget.LowImportance
		a.setPage(container.NewCenter(widget.NewCard(
			"Katalog masih kosong",
			"Tambahkan produk pertama untuk mulai berjualan",
			container.NewVBox(addFirst, backEmpty),
		)), active, "cashier")
		return
	}

	operationID, err := salesapp.NewOperationID()
	if err != nil {
		a.showError(err)
		return
	}
	search := widget.NewEntry()
	search.SetPlaceHolder("Cari nama, SKU, atau barcode")
	categoryOptions := []string{"Semua kategori"}
	categorySet := make(map[string]struct{})
	for _, item := range items {
		if category := strings.TrimSpace(item.Category); category != "" {
			categorySet[category] = struct{}{}
		}
	}
	categories := make([]string, 0, len(categorySet))
	for category := range categorySet {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	categoryOptions = append(categoryOptions, categories...)
	category := widget.NewSelect(categoryOptions, nil)
	category.SetSelected("Semua kategori")
	preorderMode := widget.NewCheck("Terima pre-order", nil)
	productCount := widget.NewLabel("")
	productGrid := container.NewGridWithColumns(maxProductColumns)
	gridHolder := container.New(&responsiveGrid{grid: productGrid}, productGrid)
	productScroll := container.NewVScroll(container.NewVBox(gridHolder))
	cartRows := container.NewVBox()
	totalLabel := widget.NewLabelWithStyle("Rp 0", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	paymentState := wrappedLabel("Tambahkan produk untuk mulai transaksi.")
	paymentLabels := []string{"Tunai", "Transfer bank", "Kartu", "E-wallet"}
	paymentMethods := map[string]string{
		"Tunai":         saledomain.PaymentCash,
		"Transfer bank": saledomain.PaymentBankTransfer,
		"Kartu":         saledomain.PaymentCard,
		"E-wallet":      saledomain.PaymentEWallet,
	}
	paymentMethod := widget.NewSelect(paymentLabels, nil)
	paymentMethod.SetSelected("Tunai")
	cashReceived := widget.NewEntry()
	cashReceived.SetPlaceHolder("Contoh: 50.000")
	cashField := container.NewVBox(widget.NewLabel("Uang diterima (Rp)"), cashReceived)
	cart := make([]cartLine, 0)
	checkout := widget.NewButton("Bayar", nil)
	checkout.Importance = widget.SuccessImportance
	checkout.Disable()
	var renderProducts func()
	var renderCart func()
	var refreshPayment func()
	var addItem func(catalogdomain.Item, bool)

	refreshPayment = func() {
		if len(cart) == 0 {
			paymentState.SetText("Tambahkan produk untuk mulai transaksi.")
			checkout.SetText("Bayar")
			checkout.Disable()
			return
		}
		total, err := cartTotal(cart)
		checkout.SetText("Bayar · " + formatRupiah(total))
		if err != nil {
			paymentState.SetText("Total transaksi terlalu besar untuk diproses.")
			checkout.Disable()
			return
		}
		if paymentMethods[paymentMethod.Selected] == saledomain.PaymentCash {
			received, err := parseRupiah(cashReceived.Text, true)
			if err != nil {
				paymentState.SetText("Masukkan jumlah uang tunai yang diterima.")
				checkout.Disable()
				return
			}
			if received < total {
				paymentState.SetText("Uang kurang " + formatRupiah(total-received))
				checkout.Disable()
				return
			}
			paymentState.SetText("Kembalian " + formatRupiah(received-total))
			checkout.Enable()
			return
		}
		paymentState.SetText("Pembayaran " + strings.ToLower(paymentMethod.Selected) + " akan dicatat manual.")
		checkout.Enable()
	}

	renderCart = func() {
		cartRows.Objects = nil
		if len(cart) == 0 {
			cartRows.Add(wrappedLabel("Keranjang masih kosong. Pilih produk di sebelah kiri untuk mulai."))
			totalLabel.SetText("Rp 0")
			refreshPayment()
			cartRows.Refresh()
			return
		}
		total, err := cartTotal(cart)
		if err != nil {
			totalLabel.SetText("Tidak dapat dihitung")
			refreshPayment()
			cartRows.Refresh()
			return
		}
		totalLabel.SetText(formatRupiah(total))
		for index, line := range cart {
			lineIndex := index
			fulfillment := ""
			if line.preOrder {
				fulfillment = " · Pre-order"
			}
			lineSubtotal := line.item.PriceRupiah * line.quantity
			name := widget.NewLabelWithStyle(line.item.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			amount := widget.NewLabelWithStyle(formatRupiah(lineSubtotal), fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
			detail := widget.NewLabel(fmt.Sprintf("%s × %d %s%s", formatRupiah(line.item.PriceRupiah), line.quantity, line.item.Unit, fulfillment))
			minus := widget.NewButton("−", func() {
				if cart[lineIndex].quantity <= 1 {
					cart = append(cart[:lineIndex], cart[lineIndex+1:]...)
				} else {
					cart[lineIndex].quantity--
				}
				renderCart()
				renderProducts()
			})
			quantityLabel := widget.NewLabel(strconv.FormatInt(line.quantity, 10))
			plus := widget.NewButton("+", func() { addItem(line.item, line.preOrder) })
			remove := widget.NewButton("Hapus", func() {
				cart = append(cart[:lineIndex], cart[lineIndex+1:]...)
				renderCart()
				renderProducts()
			})
			remove.Importance = widget.LowImportance
			stepper := container.NewHBox(minus, container.NewCenter(quantityLabel), plus)
			row := container.NewVBox(
				container.NewHBox(name, layout.NewSpacer(), amount),
				detail,
				container.NewHBox(stepper, layout.NewSpacer(), remove),
			)
			cartRows.Add(row)
			cartRows.Add(widget.NewSeparator())
		}
		refreshPayment()
		cartRows.Refresh()
	}

	addItem = func(item catalogdomain.Item, asPreorder bool) {
		if asPreorder && (!item.TracksStock || !item.AllowsPreorder) {
			a.showError(fmt.Errorf("pre-order tidak tersedia untuk %s", item.Name))
			return
		}
		for index := range cart {
			if cart[index].item.ID != item.ID {
				continue
			}
			if cart[index].preOrder != asPreorder {
				a.showError(fmt.Errorf("%s sudah ada dengan cara pemenuhan berbeda; hapus dulu untuk menggantinya", item.Name))
				return
			}
			if cart[index].quantity == math.MaxInt64 {
				a.showError(fmt.Errorf("jumlah %s sudah mencapai batas", item.Name))
				return
			}
			if item.TracksStock && !asPreorder && cart[index].quantity+1 > item.StockOnHand {
				a.showError(fmt.Errorf("stok %s tinggal %d %s", item.Name, item.StockOnHand-cart[index].quantity, item.Unit))
				return
			}
			cart[index].quantity++
			renderCart()
			renderProducts()
			return
		}
		if item.TracksStock && !asPreorder && item.StockOnHand < 1 {
			a.showError(fmt.Errorf("stok %s habis; aktifkan mode pre-order jika pesanan tetap diterima", item.Name))
			return
		}
		cart = append(cart, cartLine{item: item, quantity: 1, preOrder: asPreorder})
		renderCart()
		renderProducts()
	}

	renderProducts = func() {
		cards := make([]fyne.CanvasObject, 0, len(items))
		searchTerm := strings.ToLower(strings.TrimSpace(search.Text))
		for _, item := range items {
			product := item
			if category.Selected != "Semua kategori" && item.Category != category.Selected {
				continue
			}
			searchable := strings.ToLower(strings.Join([]string{item.Name, item.SKU, item.Barcode, item.Category}, " "))
			if searchTerm != "" && !strings.Contains(searchable, searchTerm) {
				continue
			}
			stockText := "Stok tidak dilacak"
			if item.Type == catalogdomain.TypeService {
				stockText = "Jasa · tanpa stok"
			}
			canAdd := true
			if item.TracksStock {
				remaining := item.StockOnHand - cartQuantityForStock(cart, item.ID)
				if remaining < 0 {
					remaining = 0
				}
				stockText = fmt.Sprintf("Tersedia %d %s", remaining, item.Unit)
				if preorderMode.Checked {
					if item.AllowsPreorder {
						stockText = "Pre-order · stok tidak dipotong"
					} else {
						stockText = "Tidak menerima pre-order"
						canAdd = false
					}
				} else if remaining == 0 {
					stockText = "Stok habis"
					canAdd = false
				}
			} else if preorderMode.Checked {
				stockText = "Item ini tidak mendukung pre-order"
				canAdd = false
			}
			for _, line := range cart {
				if line.item.ID == item.ID && line.preOrder != preorderMode.Checked {
					stockText = "Hapus dari keranjang untuk mengganti pemenuhan"
					canAdd = false
					break
				}
			}
			add := widget.NewButton("Tambah", func() { addItem(product, preorderMode.Checked) })
			if !canAdd {
				add.Disable()
			}
			thumb := canvas.NewRectangle(color.Transparent)
			thumb.SetMinSize(fyne.NewSize(200, 130))
			cardContent := container.NewVBox()
			if resource, ok := imageResources[item.ID]; ok {
				image := canvas.NewImageFromResource(resource)
				image.FillMode = canvas.ImageFillContain
				image.SetMinSize(fyne.NewSize(200, 130))
				cardContent.Add(image)
			} else {
				cardContent.Add(thumb)
			}
			cardContent.Add(primaryText(formatRupiah(item.PriceRupiah), 17))
			cardContent.Add(captionLabel(stockText))
			if canAdd {
				add.Importance = widget.HighImportance
			}
			cardContent.Add(add)
			cardSubtitle := item.Category
			if cardSubtitle == "" {
				cardSubtitle = "Jasa"
				if item.Type != catalogdomain.TypeService {
					cardSubtitle = "Barang"
				}
			}
			cards = append(cards, widget.NewCard(item.Name, cardSubtitle, cardContent))
		}
		productCount.SetText(fmt.Sprintf("%d produk", len(cards)))
		if len(cards) == 0 {
			productGrid.Objects = []fyne.CanvasObject{wrappedLabel("Produk tidak ditemukan. Coba kata kunci atau kategori lain.")}
		} else {
			productGrid.Objects = cards
		}
		productGrid.Refresh()
	}

	search.OnChanged = func(string) { renderProducts() }
	category.OnChanged = func(string) { renderProducts() }
	preorderMode.OnChanged = func(bool) { renderProducts() }
	cashReceived.OnChanged = func(string) { refreshPayment() }
	paymentMethod.OnChanged = func(selected string) {
		if paymentMethods[selected] == saledomain.PaymentCash {
			cashField.Show()
		} else {
			cashField.Hide()
		}
		refreshPayment()
	}
	checkout.OnTapped = func() {
		method := paymentMethods[paymentMethod.Selected]
		if method == "" {
			a.showError(fmt.Errorf("pilih metode pembayaran"))
			return
		}
		cash := int64(0)
		if method == saledomain.PaymentCash {
			var err error
			cash, err = parseRupiah(cashReceived.Text, true)
			if err != nil {
				a.showError(fmt.Errorf("masukkan jumlah uang tunai yang valid"))
				return
			}
		}
		lines := make([]saledomain.LineInput, 0, len(cart))
		for _, line := range cart {
			lines = append(lines, saledomain.LineInput{ItemID: line.item.ID, Quantity: line.quantity, PreOrder: line.preOrder})
		}
		input := saledomain.CheckoutInput{
			OperationID:        operationID,
			BusinessID:         active.ID,
			OutletID:           outlet.ID,
			Lines:              lines,
			PaymentMethod:      method,
			CashReceivedRupiah: cash,
		}
		checkoutCtx, checkoutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer checkoutCancel()
		receipt, err := a.sales.Checkout(checkoutCtx, input)
		if err != nil {
			a.showError(err)
			return
		}
		cart = nil
		cashReceived.SetText("")
		refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 5*time.Second)
		refreshedItems, refreshErr := a.catalog.List(refreshCtx, active.ID, outlet.ID)
		refreshCancel()
		if refreshErr == nil {
			items = refreshedItems
		} else {
			for _, soldLine := range receipt.Lines {
				if soldLine.Fulfillment != "stock" {
					continue
				}
				for index := range items {
					if items[index].ID == soldLine.ItemID && items[index].StockOnHand >= soldLine.Quantity {
						items[index].StockOnHand -= soldLine.Quantity
					}
				}
			}
		}
		operationID, err = salesapp.NewOperationID()
		if err != nil {
			a.showError(err)
			return
		}
		renderCart()
		renderProducts()
		receiptLines := []string{"Metode: " + paymentMethodName(receipt.PaymentMethod)}
		if receipt.PaymentMethod == saledomain.PaymentCash {
			receiptLines = append(receiptLines,
				"Tunai diterima "+formatRupiah(receipt.CashReceivedRupiah),
				"Kembalian "+formatRupiah(receipt.ChangeRupiah),
			)
		}
		if refreshErr != nil {
			receiptLines = append(receiptLines, "Stok diperbarui sementara; buka ulang kasir untuk memuat data terbaru.")
		}
		a.successDialog("Transaksi tersimpan", "TRANSAKSI BERHASIL", formatRupiah(receipt.TotalRupiah), receiptLines)
	}

	clearCart := widget.NewButton("Kosongkan", func() {
		cart = nil
		renderCart()
		renderProducts()
	})
	cartHeader := container.NewBorder(nil, hairline(), nil, container.NewVBox(clearCart), container.NewVBox(
		sectionTitle("Keranjang"),
		wrappedLabel(fmt.Sprintf("%s · %s", active.Name, outlet.Name)),
	))
	checkoutPanel := container.NewVBox(
		hairline(),
		container.NewBorder(nil, nil, nil, totalLabel, captionLabel("TOTAL BELANJA")),
		widget.NewLabel("Metode pembayaran"),
		paymentMethod,
		cashField,
		paymentState,
		wrappedLabel("Pembayaran transfer, kartu, dan e-wallet dicatat manual."),
		checkout,
		widget.NewButton("Kembali ke ringkasan", a.showHome),
	)
	cartPanel := container.NewBorder(cartHeader, checkoutPanel, nil, nil, container.NewVScroll(cartRows))
	productHeader := container.NewVBox(
		captionLabel("PRODUK"),
		container.NewHBox(sectionTitle("Pilih produk"), productCount),
		search,
		category,
		preorderMode,
		wrappedLabel("Mode pre-order melewati cek stok dan tidak mengurangi stok."),
		hairline(),
	)
	productPanel := container.NewBorder(productHeader, nil, nil, nil, productScroll)
	split := container.NewHSplit(container.NewPadded(productPanel), container.NewPadded(cartPanel))
	split.SetOffset(0.65)
	backToHome := widget.NewButton("Kembali", a.showHome)
	backToHome.Importance = widget.LowImportance
	header := container.NewBorder(nil, hairline(), nil, container.NewVBox(backToHome), container.NewVBox(
		goldText("◆ TRANSAKSI", 12),
		pageTitle("Kasir"),
		wrappedLabel(fmt.Sprintf("%s · %s", active.Name, outlet.Name)),
	))
	a.setPage(container.NewBorder(header, nil, nil, nil, split), active, "cashier")
	renderProducts()
	renderCart()
}

func cartQuantityForStock(lines []cartLine, itemID string) int64 {
	var quantity int64
	for _, line := range lines {
		if line.item.ID == itemID && !line.preOrder {
			quantity += line.quantity
		}
	}
	return quantity
}

func cartTotal(lines []cartLine) (int64, error) {
	var total int64
	for _, line := range lines {
		if line.item.PriceRupiah > 0 && line.quantity > math.MaxInt64/line.item.PriceRupiah {
			return 0, fmt.Errorf("nilai transaksi terlalu besar")
		}
		lineTotal := line.item.PriceRupiah * line.quantity
		if total > math.MaxInt64-lineTotal {
			return 0, fmt.Errorf("nilai transaksi terlalu besar")
		}
		total += lineTotal
	}
	return total, nil
}

func (a *App) showJournal(active Business) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	entries, err := a.sales.ListJournal(ctx, active.ID)
	cancel()
	if err != nil {
		a.showError(err)
		return
	}
	title := container.NewVBox(
		goldText("◆ RIWAYAT", 12),
		pageTitle("Jurnal transaksi"),
		wrappedLabel(fmt.Sprintf("%s · %d transaksi tercatat", active.Name, len(entries))),
		hairline(),
	)
	openCashier := widget.NewButton("Buka Kasir", func() { a.showCashier(active) })
	openCashier.Importance = widget.HighImportance
	backHome := widget.NewButton("Kembali ke ringkasan", a.showHome)
	backHome.Importance = widget.LowImportance
	footer := container.NewHBox(openCashier, backHome)
	if len(entries) == 0 {
		a.setPage(container.NewCenter(widget.NewCard(
			"Jurnal masih kosong",
			"Transaksi yang tersimpan akan tercatat di sini",
			container.NewVBox(openCashier, backHome),
		)), active, "journal")
		return
	}
	rows := make(map[fyne.CanvasObject]*journalRowView)
	list := widget.NewList(
		func() int { return len(entries) },
		func() fyne.CanvasObject {
			badge := canvas.NewRectangle(posPrimaryColor)
			badge.SetMinSize(fyne.NewSize(4, 44))
			view := &journalRowView{
				badge:  badge,
				name:   widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				date:   captionLabel(""),
				amount: widget.NewLabelWithStyle("", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
				ref:    captionLabel(""),
			}
			view.ref.Alignment = fyne.TextAlignTrailing
			row := container.NewHBox(
				badge,
				container.NewVBox(view.name, view.date),
				layout.NewSpacer(),
				container.NewVBox(view.amount, view.ref),
			)
			padded := container.NewPadded(row)
			rows[padded] = view
			return padded
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			entry := entries[id]
			view, ok := rows[object]
			if !ok {
				return
			}
			view.name.SetText(journalEventName(entry.EventType))
			view.date.Text = entry.OccurredAt.Local().Format("02 Jan 2006 · 15:04")
			view.date.Refresh()
			view.amount.SetText(formatRupiah(entry.AmountRupiah))
			view.ref.Text = "#" + shortID(entry.SourceID)
			view.ref.Alignment = fyne.TextAlignTrailing
			view.ref.Refresh()
			view.badge.FillColor = journalEventColor(entry.EventType)
			view.badge.Refresh()
		},
	)
	a.setPage(container.NewBorder(title, footer, nil, nil, list), active, "journal")
}

func firstOutlet(active Business) (Outlet, bool) {
	if len(active.Outlets) == 0 {
		return Outlet{}, false
	}
	return active.Outlets[0], true
}

func parseRupiah(text string, allowEmpty bool) (int64, error) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(text), ".", "")
	if cleaned == "" && allowEmpty {
		return 0, nil
	}
	value, err := strconv.ParseInt(cleaned, 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("masukkan angka rupiah yang valid")
	}
	return value, nil
}

func formatRupiah(value int64) string {
	digits := strconv.FormatInt(value, 10)
	var grouped strings.Builder
	for index, digit := range digits {
		if index > 0 && (len(digits)-index)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(digit)
	}
	return "Rp " + grouped.String()
}

func shortID(value string) string {
	if len(value) > 6 {
		return value[:6]
	}
	return value
}

// productRowView keeps references to a catalog list row template so the
// update function can fill texts and the thumbnail without searching.
type productRowView struct {
	image *canvas.Image
	blank *canvas.Rectangle
	name  *widget.Label
	sub   *canvas.Text
	price *canvas.Text
	stock *canvas.Text
}

// journalRowView keeps references to a journal list row template so the
// update function can fill texts and the event badge without searching.
type journalRowView struct {
	badge  *canvas.Rectangle
	name   *widget.Label
	date   *canvas.Text
	amount *widget.Label
	ref    *canvas.Text
}

// journalEventColor marks ledger rows: emerald for sales, gold for
// payments, gray for stock movements.
func journalEventColor(eventType string) color.Color {
	switch eventType {
	case "sale":
		return posPrimaryColor
	case "payment":
		return posGoldColor
	default:
		return color.NRGBA{R: 150, G: 158, B: 153, A: 255}
	}
}

func journalEventName(eventType string) string {
	switch eventType {
	case "sale":
		return "Penjualan"
	case "payment":
		return "Pembayaran"
	case "stock_opening":
		return "Stok awal"
	case "stock_restock":
		return "Stok masuk"
	case "stock_adjustment":
		return "Penyesuaian stok"
	default:
		return eventType
	}
}

func paymentMethodName(method string) string {
	switch method {
	case saledomain.PaymentCash:
		return "Tunai"
	case saledomain.PaymentBankTransfer:
		return "Transfer bank"
	case saledomain.PaymentCard:
		return "Kartu"
	case saledomain.PaymentEWallet:
		return "E-wallet"
	default:
		return method
	}
}
