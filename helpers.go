package secureentry

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// currentVariant returns the active theme variant, falling back to dark when
// no app (and thus no settings) is available.
func currentVariant() fyne.ThemeVariant {
	if a := fyne.CurrentApp(); a != nil {
		return a.Settings().ThemeVariant()
	}
	return theme.VariantDark
}

// measureText measures a text string for display purposes. It is only ever
// called with strings that contain no sensitive content (mask glyphs and
// placeholders), never with the actual input.
func measureText(text string, size float32, style fyne.TextStyle) fyne.Size {
	if a := fyne.CurrentApp(); a != nil {
		sz, _ := a.Driver().RenderedTextSize(text, size, style, nil)
		return sz
	}
	return fyne.Size{}
}

// themePrimaryColor returns the primary colour for the active variant.
func themePrimaryColor() color.Color {
	return theme.Current().Color(theme.ColorNamePrimary, currentVariant())
}