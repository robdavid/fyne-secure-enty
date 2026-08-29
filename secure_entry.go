// Package secureentry provides a Fyne text entry widget designed for secure
// data such as passwords.
//
// SecureEntry stores the user's input exclusively in a fixed-length, zeroable
// []byte buffer. The content is never converted to a Go string: Go strings are
// immutable and cannot be erased, so they would leave the input retained in
// memory until garbage collection. Everything the widget renders on screen is
// derived from a single non-sensitive mask rune repeated RuneCount times, so
// the actual content never leaks into the display layer either.
//
// The contract for callers:
//
//   - Read the input with Bytes() and process it purely as a byte slice.
//   - Do not convert Bytes() to a string.
//   - Call Erase() once the input has been processed to wipe the buffer from
//     memory.
//
// Editing is intentionally minimal (append-only): characters can be appended
// and the last rune can be removed with Backspace. Cursor navigation, text
// selection and the clipboard (paste/copy/cut/select-all) and undo/redo are
// deliberately disabled so that no copy of the content is ever created,
// exported or retained beyond the single internal buffer.
package secureentry

import (
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"
)

// DefaultMaskRune is the glyph used to conceal the entry content. The Unicode
// bullet is widely supported by the fonts bundled with Fyne.
const DefaultMaskRune = '\u2022' // '•'

// SecureEntry is a secure text entry widget whose content lives only in a
// fixed-length, erasable byte slice.
type SecureEntry struct {
	widget.DisableableWidget

	buffer    []byte
	count     int
	runeCount int

	maskRune rune

	PlaceHolder string
	OnChanged   func([]byte) `json:"-"`
	OnSubmitted func([]byte) `json:"-"`

	focused bool
	cursor  *cursorAnim
}

var (
	_ fyne.Widget         = (*SecureEntry)(nil)
	_ fyne.Focusable      = (*SecureEntry)(nil)
	_ fyne.Tappable       = (*SecureEntry)(nil)
	_ fyne.Shortcutable   = (*SecureEntry)(nil)
	_ fyne.Disableable    = (*SecureEntry)(nil)
	_ fyne.Tabbable       = (*SecureEntry)(nil)
	_ mobile.Keyboardable = (*SecureEntry)(nil)
	_ desktop.Cursorable  = (*SecureEntry)(nil)
)

// NewSecureEntry creates a SecureEntry with an input buffer of maxLen bytes.
// The buffer is allocated once and never reallocated; it is the only location
// in which the user's input is ever stored.
func NewSecureEntry(maxLen int) *SecureEntry {
	if maxLen < 0 {
		maxLen = 0
	}
	e := &SecureEntry{
		buffer:   make([]byte, maxLen),
		maskRune: DefaultMaskRune,
	}
	e.ExtendBaseWidget(e)
	return e
}

// Bytes returns the current content as a byte slice backed by the internal
// fixed-length buffer. The returned slice is only valid until the next input
// event or call to Erase. Process the input as a byte slice and call Erase()
// when it is no longer needed.
func (e *SecureEntry) Bytes() []byte { return e.buffer[:e.count] }

// Len returns the number of content bytes currently held.
func (e *SecureEntry) Len() int { return e.count }

// RuneCount returns the number of characters (runes) currently held. This is
// the number of mask glyphs displayed.
func (e *SecureEntry) RuneCount() int { return e.runeCount }

// Erase wipes the entire input buffer from memory, resets the content length
// to zero and clears the display. It is safe to call at any time, including
// after processing the value returned by Bytes.
func (e *SecureEntry) Erase() {
	clear(e.buffer)
	e.count = 0
	e.runeCount = 0
	e.Refresh()
}

// SetMaskRune sets the character used to conceal the content. The default is
// the Unicode bullet '•'.
func (e *SecureEntry) SetMaskRune(r rune) {
	if r == 0 {
		r = DefaultMaskRune
	}
	e.maskRune = r
	e.Refresh()
}

// SetPlaceHolder sets the text displayed while the entry is empty.
func (e *SecureEntry) SetPlaceHolder(text string) {
	e.PlaceHolder = text
	e.Refresh()
}

// CreateRenderer links this widget to its renderer.
func (e *SecureEntry) CreateRenderer() fyne.WidgetRenderer {
	e.ExtendBaseWidget(e)
	return newRenderer(e)
}

// AcceptsTab reports that this entry does not accept the Tab key.
func (e *SecureEntry) AcceptsTab() bool { return false }

// Keyboard returns the password keyboard so mobile IMEs offer no suggestions.
func (e *SecureEntry) Keyboard() mobile.KeyboardType {
	return mobile.PasswordKeyboard
}

// Cursor returns the desktop pointer icon for this widget.
func (e *SecureEntry) Cursor() desktop.Cursor {
	return desktop.TextCursor
}

// Tapped requests keyboard focus when the widget is clicked.
func (e *SecureEntry) Tapped(_ *fyne.PointEvent) {
	if e.Disabled() {
		return
	}
	e.requestFocus()
}

// FocusGained marks the widget as focused and refreshes the cursor display.
func (e *SecureEntry) FocusGained() {
	e.focused = true
	e.Refresh()
}

// FocusLost clears the focused state and refreshes the cursor display.
func (e *SecureEntry) FocusLost() {
	e.focused = false
	e.Refresh()
}

// TypedRune appends the UTF-8 encoding of r to the internal buffer. The rune
// is encoded into a stack array and copied into the fixed buffer; no string is
// ever constructed.
func (e *SecureEntry) TypedRune(r rune) {
	if e.Disabled() || r == 0 {
		return
	}
	if e.cursor != nil {
		e.cursor.interrupt()
	}
	if e.insert(r) {
		e.Refresh()
	}
}

// TypedKey handles Backspace (removes the last rune) and Enter/Return
// (triggers OnSubmitted). All navigation keys are ignored: this is an
// append-only secure field.
func (e *SecureEntry) TypedKey(key *fyne.KeyEvent) {
	if e.Disabled() {
		return
	}
	if e.cursor != nil {
		e.cursor.interrupt()
	}
	switch key.Name {
	case fyne.KeyBackspace:
		if e.removeLast() {
			e.Refresh()
		}
	case fyne.KeyReturn, fyne.KeyEnter:
		if e.OnSubmitted != nil {
			e.OnSubmitted(e.Bytes())
		}
	default:
		// Deliberately ignored: cursor movement, undo/redo etc. must not
		// mutate or expose the secure content.
	}
}

// TypedShortcut deliberately ignores every shortcut (paste, copy, cut,
// select-all, undo, redo). Clipboard content is never imported and no copy of
// the content is ever created or exported.
func (e *SecureEntry) TypedShortcut(fyne.Shortcut) {
}

// requestFocus moves the canvas focus to this widget.
func (e *SecureEntry) requestFocus() {
	if c := fyne.CurrentApp().Driver().CanvasForObject(e); c != nil {
		c.Focus(e)
	}
}

// insert appends the UTF-8 encoding of r to the buffer. It reports whether the
// rune was accepted, which is false when the buffer cannot hold it.
func (e *SecureEntry) insert(r rune) bool {
	if e.count == len(e.buffer) {
		return false
	}
	var enc [utf8.UTFMax]byte
	n := utf8.EncodeRune(enc[:], r)
	end := e.count + n
	if end > len(e.buffer) {
		return false
	}
	copy(e.buffer[e.count:], enc[:n])
	e.count = end
	e.runeCount++
	if e.OnChanged != nil {
		e.OnChanged(e.Bytes())
	}
	return true
}

// removeLast removes the final rune from the buffer and zeroes the bytes it
// freed. It reports whether anything was removed.
func (e *SecureEntry) removeLast() bool {
	if e.count == 0 {
		return false
	}
	_, size := utf8.DecodeLastRune(e.buffer[:e.count])
	start := e.count - size
	e.count = start
	e.runeCount--
	clear(e.buffer[start : start+size])
	if e.OnChanged != nil {
		e.OnChanged(e.Bytes())
	}
	return true
}