package fyneui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
)

// Luxury light palette: deep emerald primary, champagne gold accents,
// cool neutral surfaces. Always light regardless of OS setting.
// No gradients, single radius system.
var (
	posPrimaryColor    = color.NRGBA{R: 13, G: 90, B: 69, A: 255}   // deep emerald
	posBackgroundColor = color.NRGBA{R: 236, G: 238, B: 239, A: 255} // cool light gray
	posSurfaceColor    = color.NRGBA{R: 255, G: 255, B: 255, A: 255} // cards, sidebar
	posHeaderColor     = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	posInkColor        = color.NRGBA{R: 28, G: 36, B: 33, A: 255}
	posMutedColor      = color.NRGBA{R: 95, G: 107, B: 102, A: 255}
	posBorderColor     = color.NRGBA{R: 220, G: 225, B: 221, A: 255}
	posButtonColor     = color.NRGBA{R: 231, G: 236, B: 232, A: 255}
	posHoverColor      = color.NRGBA{R: 214, G: 229, B: 220, A: 255}
	posPressedColor    = color.NRGBA{R: 202, G: 219, B: 209, A: 255}
	posSelectionColor  = color.NRGBA{R: 213, G: 228, B: 219, A: 255}

	posGoldColor     = color.NRGBA{R: 194, G: 163, B: 94, A: 255} // champagne gold
	posGoldDarkColor = color.NRGBA{R: 150, G: 119, B: 60, A: 255} // gold text on light
	posBrandDark     = color.NRGBA{R: 17, G: 27, B: 23, A: 255}   // brand plate
	posWarmWhite     = color.NRGBA{R: 244, G: 241, B: 232, A: 255} // text on dark
	posDarkMuted     = color.NRGBA{R: 158, G: 168, B: 163, A: 255} // muted on dark
	posErrorColor    = color.NRGBA{R: 179, G: 54, B: 43, A: 255}
)

type posTheme struct{}

var _ fyne.Theme = (*posTheme)(nil)

// Color always returns the luxury light palette, ignoring the OS variant,
// so the app stays light on every system.
func (posTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary, theme.ColorNameFocus:
		return posPrimaryColor
	case theme.ColorNameBackground:
		return posBackgroundColor
	case theme.ColorNameForeground:
		return posInkColor
	case theme.ColorNameHeaderBackground, theme.ColorNameMenuBackground:
		return posHeaderColor
	case theme.ColorNameInputBackground:
		return posSurfaceColor
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return posBorderColor
	case theme.ColorNameButton:
		return posButtonColor
	case theme.ColorNameHover:
		return posHoverColor
	case theme.ColorNamePressed:
		return posPressedColor
	case theme.ColorNameSelection:
		return posSelectionColor
	case theme.ColorNameSuccess:
		return posPrimaryColor
	case theme.ColorNameWarning:
		return posGoldDarkColor
	case theme.ColorNameError:
		return posErrorColor
	case theme.ColorNameHyperlink:
		return posPrimaryColor
	case theme.ColorNamePlaceHolder, theme.ColorNameDisabled:
		return posMutedColor
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 228, G: 232, B: 229, A: 255}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 190, G: 198, B: 193, A: 255}
	case theme.ColorNameForegroundOnPrimary,
		theme.ColorNameForegroundOnSuccess,
		theme.ColorNameForegroundOnWarning,
		theme.ColorNameForegroundOnError:
		return color.White
	}
	return theme.DefaultTheme().Color(name, theme.VariantLight)
}

func (posTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (posTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (posTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 24
	case theme.SizeNameSubHeadingText:
		return 17
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNamePadding:
		return 12
	case theme.SizeNameInnerPadding:
		return 10
	case theme.SizeNameInputRadius:
		return 8
	case theme.SizeNameButtonRadius:
		return 8
	case theme.SizeNameCardRadius:
		return 12
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameInputBorder:
		return 1
	case theme.SizeNameLineSpacing:
		return 4
	}
	return theme.DefaultTheme().Size(name)
}

// ---- Luxury helpers: single-line display text on known surfaces. ----

func brandTitle(text string) *canvas.Text {
	t := canvas.NewText(text, posWarmWhite)
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = 18
	return t
}

func brandCaption(text string) *canvas.Text {
	t := canvas.NewText(text, posDarkMuted)
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = 11
	return t
}

func pageTitle(text string) *canvas.Text {
	t := canvas.NewText(text, posInkColor)
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = 22
	return t
}

func sectionTitle(text string) *canvas.Text {
	t := canvas.NewText(text, posInkColor)
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = 16
	return t
}

// captionLabel renders a short uppercase grouping label, e.g. "MENU".
func captionLabel(text string) *canvas.Text {
	t := canvas.NewText(text, posMutedColor)
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = 11
	return t
}

func goldText(text string, size float32) *canvas.Text {
	t := canvas.NewText(text, posGoldColor)
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = size
	return t
}

func primaryText(text string, size float32) *canvas.Text {
	t := canvas.NewText(text, posPrimaryColor)
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = size
	return t
}

// goldRule is a full-width champagne hairline used under brand blocks.
func goldRule() *canvas.Rectangle {
	r := canvas.NewRectangle(posGoldColor)
	r.SetMinSize(fyne.NewSize(1, 2))
	return r
}

// hairline is a thin neutral divider for light surfaces.
func hairline() *canvas.Rectangle {
	r := canvas.NewRectangle(posBorderColor)
	r.SetMinSize(fyne.NewSize(1, 1))
	return r
}
