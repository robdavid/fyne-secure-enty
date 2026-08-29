package secureentry

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
)

type secureEntryRenderer struct {
	entry       *SecureEntry
	box         *canvas.Rectangle
	border      *canvas.Rectangle
	text        *canvas.Text
	placeholder *canvas.Text
	cursor      *canvas.Rectangle
	objects     []fyne.CanvasObject
}

func newRenderer(e *SecureEntry) fyne.WidgetRenderer {
	th := e.Theme()
	borderWidth := th.Size(theme.SizeNameInputBorder)
	border := canvas.NewRectangle(nil)
	border.StrokeWidth = borderWidth
	border.CornerRadius = th.Size(theme.SizeNameInputRadius)
	box := canvas.NewRectangle(nil)
	box.CornerRadius = border.CornerRadius

	textSize := th.Size(theme.SizeNameText)
	text := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	text.TextSize = textSize
	placeholder := canvas.NewText(e.PlaceHolder, theme.Color(theme.ColorNamePlaceHolder))
	placeholder.TextSize = textSize

	cursor := canvas.NewRectangle(theme.Color(theme.ColorNamePrimary))
	cursor.Hide()

	r := &secureEntryRenderer{
		entry:       e,
		box:         box,
		border:      border,
		text:        text,
		placeholder: placeholder,
		cursor:      cursor,
		objects:     []fyne.CanvasObject{box, border, placeholder, text, cursor},
	}

	e.cursor = newCursorAnim(cursor)
	r.update()
	return r
}

var _ fyne.WidgetRenderer = (*secureEntryRenderer)(nil)

func (r *secureEntryRenderer) Destroy() {
	r.entry.cursor.stop()
}

func (r *secureEntryRenderer) Layout(size fyne.Size) {
	th := r.entry.Theme()
	borderSize := th.Size(theme.SizeNameInputBorder)

	r.border.Resize(fyne.NewSize(size.Width-borderSize, size.Height-borderSize))
	r.border.Move(fyne.NewPos(borderSize/2, borderSize/2))
	r.box.Resize(fyne.NewSize(size.Width-borderSize*2, size.Height-borderSize*2))
	r.box.Move(fyne.NewPos(borderSize, borderSize))

	x, y, w, h := r.textGeometry(size)
	r.text.Resize(fyne.NewSize(w, h))
	r.text.Move(fyne.NewPos(x, y))
	r.placeholder.Resize(fyne.NewSize(w, h))
	r.placeholder.Move(fyne.NewPos(x, y))
	r.positionCursor()
}

func (r *secureEntryRenderer) MinSize() fyne.Size {
	th := r.entry.Theme()
	borderSize := th.Size(theme.SizeNameInputBorder)
	textSize := th.Size(theme.SizeNameText)
	style := fyne.TextStyle{}

	charSize := measureText("M", textSize, style)
	width := charSize.Width * 6
	if ph := measureText(r.entry.PlaceHolder, textSize, style).Width; ph > width {
		width = ph
	}
	width += borderSize * 2
	height := charSize.Height + borderSize*2
	return fyne.NewSize(width, height)
}

func (r *secureEntryRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *secureEntryRenderer) Refresh() {
	r.update()

	r.box.Refresh()
	r.border.Refresh()
	r.text.Refresh()
	r.placeholder.Refresh()

	focused := r.entry.focused && !r.entry.Disabled()
	if focused {
		settings := fyne.CurrentApp().Settings()
		if settings.ShowAnimations() {
			r.entry.cursor.start()
		} else {
			r.cursor.FillColor = theme.Color(theme.ColorNamePrimary)
		}
		r.positionCursor()
		r.cursor.Show()
	} else {
		r.entry.cursor.stop()
		r.cursor.Hide()
	}
	r.cursor.Refresh()
}

// update syncs colours, the mask/placeholder strings and object visibility
// with the widget state. None of the strings involved contain the content.
func (r *secureEntryRenderer) update() {
	e := r.entry
	th := e.Theme()
	v := currentVariant()
	focused := e.focused && !e.Disabled()

	r.box.FillColor = th.Color(theme.ColorNameInputBackground, v)
	r.box.CornerRadius = th.Size(theme.SizeNameInputRadius)
	r.border.CornerRadius = r.box.CornerRadius
	switch {
	case focused:
		r.border.StrokeColor = th.Color(theme.ColorNamePrimary, v)
	case e.Disabled():
		r.border.StrokeColor = th.Color(theme.ColorNameDisabled, v)
	default:
		r.border.StrokeColor = th.Color(theme.ColorNameInputBorder, v)
	}

	if e.Disabled() {
		r.text.Color = th.Color(theme.ColorNameDisabled, v)
		r.placeholder.Color = th.Color(theme.ColorNameDisabled, v)
	} else {
		r.text.Color = th.Color(theme.ColorNameForeground, v)
		r.placeholder.Color = th.Color(theme.ColorNamePlaceHolder, v)
	}

	if e.count == 0 {
		r.text.Text = ""
		r.placeholder.Text = e.PlaceHolder
		r.text.Hide()
		r.placeholder.Show()
	} else {
		// The mask string is derived from the character count only, so it
		// contains none of the user's sensitive input.
		r.text.Text = strings.Repeat(string(e.maskRune), e.runeCount)
		r.text.Show()
		r.placeholder.Hide()
	}
}

// positionCursor places the cursor rectangle immediately after the rendered
// mask or placeholder text.
func (r *secureEntryRenderer) positionCursor() {
	th := r.entry.Theme()
	borderSize := th.Size(theme.SizeNameInputBorder)
	textSize := th.Size(theme.SizeNameText)
	style := fyne.TextStyle{}

	x, y, _, h := r.textGeometry(r.entry.Size())
	display := r.text.Text
	if r.placeholder.Visible() {
		display = r.placeholder.Text
	}
	advance := float32(0)
	if display != "" {
		advance = measureText(display, textSize, style).Width
	}
	r.cursor.Resize(fyne.NewSize(borderSize, h))
	r.cursor.Move(fyne.NewPos(x+advance, y))
}

// textGeometry returns the x, y, width and height of the text area: inset by
// the input border width, sized for a single line of text vertically centred.
func (r *secureEntryRenderer) textGeometry(size fyne.Size) (x, y, w, h float32) {
	th := r.entry.Theme()
	borderSize := th.Size(theme.SizeNameInputBorder)
	textSize := th.Size(theme.SizeNameText)

	h = measureText("M", textSize, fyne.TextStyle{}).Height
	w = size.Width - borderSize*2
	if w < 0 {
		w = 0
	}
	x = borderSize
	y = borderSize + (size.Height-borderSize*2-h)/2
	if y < borderSize {
		y = borderSize
	}
	return x, y, w, h
}