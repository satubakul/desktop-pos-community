package fyneui

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	application "github.com/satubakul/desktop-pos-community/internal/application/business"
	domain "github.com/satubakul/desktop-pos-community/internal/domain/business"
)

func (a *App) showStartup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businesses, err := a.businesses.List(ctx)
	if err != nil {
		a.showError(err)
		return
	}
	if len(businesses) == 0 {
		a.showWelcome()
		return
	}

	activeID, err := a.businesses.ActiveID(ctx)
	if err != nil {
		a.showError(err)
		return
	}
	if !containsBusiness(businesses, activeID) {
		if err := a.businesses.SelectActive(ctx, businesses[0].ID); err != nil {
			a.showError(err)
			return
		}
	}
	a.showHome()
}

func (a *App) showWelcome() {
	width := canvas.NewRectangle(posSurfaceColor)
	width.SetMinSize(fyne.NewSize(440, 1))
	start := widget.NewButton("Mulai sekarang", a.showOnboarding)
	start.Importance = widget.HighImportance
	card := widget.NewCard("", "", container.NewVBox(
		width,
		goldText("◆ PREMIUM POINT OF SALE", 12),
		pageTitle("Kasir Kita"),
		wrappedLabel("Satu aplikasi elegan untuk mengelola beberapa usaha. Mulai dengan mengisi detail usaha dan memilih profilnya."),
		start,
		wrappedLabel("Data tersimpan lokal di perangkat ini."),
	))
	a.window.SetContent(container.NewPadded(container.NewCenter(card)))
}

func (a *App) showOnboarding() {
	name := widget.NewEntry()
	name.SetPlaceHolder("Contoh: Toko Komputer Maju")
	phone := widget.NewEntry()
	phone.SetPlaceHolder("Nomor telepon usaha")
	address := widget.NewEntry()
	address.SetPlaceHolder("Alamat usaha")
	outletName := widget.NewEntry()
	outletName.SetPlaceHolder("Contoh: Toko Pusat")
	outletAddress := widget.NewEntry()
	outletAddress.SetPlaceHolder("Alamat outlet, jika berbeda")
	customProfile := widget.NewEntry()
	customProfile.SetPlaceHolder("Isi nama profil usaha")

	options := domain.ProfileOptions()
	primaryLabels := make([]string, 0, len(options))
	secondaryLabels := make([]string, 0, len(options)-1)
	for _, option := range options {
		primaryLabels = append(primaryLabels, option.Name)
		if option.Key != "other" {
			secondaryLabels = append(secondaryLabels, option.Name)
		}
	}
	primary := widget.NewSelect(primaryLabels, nil)
	primary.PlaceHolder = "Pilih profil utama"
	secondary := widget.NewCheckGroup(secondaryLabels, nil)

	form := widget.NewForm(
		widget.NewFormItem("Nama usaha *", name),
		widget.NewFormItem("Nomor telepon", phone),
		widget.NewFormItem("Alamat usaha", address),
		widget.NewFormItem("Profil utama *", primary),
		widget.NewFormItem("Profil tambahan", secondary),
		widget.NewFormItem("Nama profil jika memilih Usaha lainnya", customProfile),
		widget.NewFormItem("Outlet pertama *", outletName),
		widget.NewFormItem("Alamat outlet", outletAddress),
	)
	form.SubmitText = "Simpan usaha"
	form.OnSubmit = func() {
		profiles := selectedProfiles(primary.Selected, secondary.Selected, customProfile.Text)
		input := application.CreateInput{
			Name:       name.Text,
			Phone:      phone.Text,
			Address:    address.Text,
			OutletName: outletName.Text,
			OutletAddr: outletAddress.Text,
			Profiles:   profiles,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := a.businesses.Create(ctx, input); err != nil {
			a.showError(err)
			return
		}
		a.showHome()
	}

	back := widget.NewButton("Kembali", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		businesses, err := a.businesses.List(ctx)
		if err != nil {
			a.showError(err)
			return
		}
		if len(businesses) == 0 {
			a.showWelcome()
			return
		}
		a.showHome()
	})
	back.Importance = widget.LowImportance
	form.OnCancel = func() { back.OnTapped() }

	header := container.NewVBox(
		goldText("◆ USAHA BARU", 12),
		pageTitle("Detail usaha"),
		wrappedLabel("Isi profil untuk usaha ini. Kamu bisa memilih profil tambahan jika usaha memiliki beberapa jenis layanan."),
		hairline(),
	)
	content := container.NewVBox(
		header,
		widget.NewCard("", "", container.NewVBox(form)),
		back,
	)
	a.window.SetContent(container.NewPadded(container.NewVScroll(content)))
}

func (a *App) showHome() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businesses, err := a.businesses.List(ctx)
	if err != nil {
		a.showError(err)
		return
	}
	if len(businesses) == 0 {
		a.showWelcome()
		return
	}
	activeID, err := a.businesses.ActiveID(ctx)
	if err != nil {
		a.showError(err)
		return
	}
	active, ok := findBusiness(businesses, activeID)
	if !ok {
		active = businesses[0]
		if err := a.businesses.SelectActive(ctx, active.ID); err != nil {
			a.showError(err)
			return
		}
	}

	labels := make([]string, 0, len(businesses))
	labelIDs := make(map[string]string, len(businesses))
	activeLabel := ""
	for _, item := range businesses {
		label := workspaceLabel(item)
		labels = append(labels, label)
		labelIDs[label] = item.ID
		if item.ID == active.ID {
			activeLabel = label
		}
	}
	selector := widget.NewSelect(labels, func(label string) {
		id := labelIDs[label]
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
	selector.SetSelected(activeLabel)

	openCashier := widget.NewButton("Mulai transaksi  →", func() { a.showCashier(active) })
	openCashier.Importance = widget.HighImportance
	header := container.NewBorder(
		nil, hairline(), nil,
		container.NewVBox(captionLabel("USAHA AKTIF"), selector),
		container.NewVBox(
			goldText("◆ RINGKASAN USAHA", 12),
			pageTitle(active.Name),
			wrappedLabel(profileSummary(active.Profiles)),
			wrappedLabel(outletSummary(active.Outlets)),
		),
	)
	hero := widget.NewCard("Kasir", "Catat penjualan barang atau jasa", container.NewVBox(
		wrappedLabel("Pilih produk, atur jumlah, lalu selesaikan pembayaran."),
		openCashier,
	))
	catalogButton := widget.NewButton("Kelola katalog", func() { a.showCatalog(active) })
	stockButton := widget.NewButton("Kelola stok", func() { a.showStock(active) })
	journalButton := widget.NewButton("Lihat jurnal", func() { a.showJournal(active) })
	addButton := widget.NewButton("Tambah usaha", a.showOnboarding)
	actions := container.NewGridWithColumns(2,
		widget.NewCard("Katalog", "Barang dan jasa", catalogButton),
		widget.NewCard("Stok", "Stok masuk dan penyesuaian", stockButton),
		widget.NewCard("Jurnal transaksi", "Riwayat penjualan dan pembayaran", journalButton),
		widget.NewCard("Usaha", "Kelola beberapa usaha", addButton),
	)
	a.setPage(container.NewVScroll(container.NewVBox(
		header,
		captionLabel("TRANSAKSI"),
		hero,
		captionLabel("KELOLA"),
		actions,
	)), active, "home")
}

func (a *App) showCashier(active Business) {
	a.showCheckout(active)
}

func (a *App) showError(err error) {
	dialog.ShowError(err, a.window)
}

// successDialog shows a roomy branded success dialog instead of Fyne's tiny
// default information box: gold caption, bold headline, detail lines.
func (a *App) successDialog(title, caption, headline string, lines []string) {
	width := canvas.NewRectangle(color.Transparent)
	width.SetMinSize(fyne.NewSize(400, 1))
	body := container.NewVBox(
		width,
		goldText("◆ "+caption, 12),
		primaryText(headline, 26),
		hairline(),
	)
	for _, line := range lines {
		body.Add(wrappedLabel(line))
	}
	dialog.ShowCustom(title, "Selesai", container.NewPadded(body), a.window)
}

func selectedProfiles(primaryLabel string, secondaryLabels []string, customName string) []Profile {
	options := domain.ProfileOptions()
	byName := make(map[string]domain.ProfileOption, len(options))
	for _, option := range options {
		byName[option.Name] = option
	}
	primary, ok := byName[primaryLabel]
	if !ok {
		return nil
	}
	primaryName := primary.Name
	if primary.Key == "other" && strings.TrimSpace(customName) != "" {
		primaryName = strings.TrimSpace(customName)
	}
	profiles := []Profile{{Key: primary.Key, Name: primaryName, Primary: true}}
	for _, label := range secondaryLabels {
		option, ok := byName[label]
		if !ok || option.Key == primary.Key {
			continue
		}
		profiles = append(profiles, Profile{Key: option.Key, Name: option.Name})
	}
	return profiles
}

func profileSummary(profiles []Profile) string {
	var names []string
	for _, profile := range profiles {
		if profile.Primary {
			names = append([]string{"Utama: " + profile.Name}, names...)
		} else {
			names = append(names, profile.Name)
		}
	}
	if len(names) == 0 {
		return "Profil usaha belum diisi."
	}
	return "Profil: " + strings.Join(names, " · ")
}

func outletSummary(outlets []Outlet) string {
	if len(outlets) == 0 {
		return "Outlet pertama belum tersedia."
	}
	names := make([]string, 0, len(outlets))
	for _, outlet := range outlets {
		names = append(names, outlet.Name)
	}
	return "Outlet: " + strings.Join(names, ", ")
}

func workspaceLabel(item Business) string {
	shortID := item.ID
	if len(shortID) > 6 {
		shortID = shortID[:6]
	}
	return fmt.Sprintf("%s · %s", item.Name, shortID)
}

func containsBusiness(items []Business, id string) bool {
	_, ok := findBusiness(items, id)
	return ok
}

func findBusiness(items []Business, id string) (Business, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return Business{}, false
}

func wrappedLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}
