package secureentry

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
)

func TestMain(m *testing.M) {
	a := test.NewApp()
	code := m.Run()
	a.Quit()
	os.Exit(code)
}

func TestTypeAndBytes(t *testing.T) {
	e := NewSecureEntry(32)
	for _, r := range "hunter2" {
		e.TypedRune(r)
	}
	if got := e.Bytes(); !bytes.Equal(got, []byte("hunter2")) {
		t.Fatalf("unexpected content: %q", got)
	}
	if e.Len() != 7 {
		t.Fatalf("Len() = %d, want 7", e.Len())
	}
	if e.RuneCount() != 7 {
		t.Fatalf("RuneCount() = %d, want 7", e.RuneCount())
	}
}

func TestBackspaceZeroesFreedBytes(t *testing.T) {
	e := NewSecureEntry(8)
	for _, r := range "abc" {
		e.TypedRune(r)
	}
	if e.buffer[2] != 'c' {
		t.Fatalf("buffer[2] = %q, want 'c'", e.buffer[2])
	}
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if got := e.Bytes(); !bytes.Equal(got, []byte("ab")) {
		t.Fatalf("content after backspace = %q, want \"ab\"", got)
	}
	if e.buffer[2] != 0 {
		t.Fatalf("freed byte not zeroed: buffer[2] = %q, want 0", e.buffer[2])
	}
	if e.Len() != 2 || e.RuneCount() != 2 {
		t.Fatalf("Len/RuneCount = %d/%d, want 2/2", e.Len(), e.RuneCount())
	}

	// Backspacing an empty field must not panic or mutate.
	e = NewSecureEntry(8)
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if e.Len() != 0 {
		t.Fatalf("empty backspace changed Len() to %d", e.Len())
	}
}

func TestAppendOnlyIgnoresNavigation(t *testing.T) {
	e := NewSecureEntry(32)
	for _, r := range "hello" {
		e.TypedRune(r)
	}
	ignored := []fyne.KeyName{
		fyne.KeyDelete, fyne.KeyLeft, fyne.KeyRight, fyne.KeyHome, fyne.KeyEnd,
		fyne.KeyUp, fyne.KeyDown, fyne.KeyPageUp, fyne.KeyPageDown,
	}
	for _, name := range ignored {
		e.TypedKey(&fyne.KeyEvent{Name: name})
	}
	if got := e.Bytes(); !bytes.Equal(got, []byte("hello")) {
		t.Fatalf("content after navigation keys = %q, want \"hello\"", got)
	}
}

func TestOnChangedCallback(t *testing.T) {
	e := NewSecureEntry(16)
	var got []byte
	e.OnChanged = func(b []byte) { got = append([]byte(nil), b...) }

	e.TypedRune('x')
	if !bytes.Equal(got, []byte("x")) {
		t.Fatalf("OnChanged after 'x' = %q, want \"x\"", got)
	}
	e.TypedRune('y')
	if !bytes.Equal(got, []byte("xy")) {
		t.Fatalf("OnChanged after 'y' = %q, want \"xy\"", got)
	}
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if !bytes.Equal(got, []byte("x")) {
		t.Fatalf("OnChanged after backspace = %q, want \"x\"", got)
	}
}

func TestOnSubmittedCallback(t *testing.T) {
	e := NewSecureEntry(16)
	var submitted []byte
	e.OnSubmitted = func(b []byte) { submitted = append([]byte(nil), b...) }

	for _, r := range "pw" {
		e.TypedRune(r)
	}
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if !bytes.Equal(submitted, []byte("pw")) {
		t.Fatalf("OnSubmitted = %q, want \"pw\"", submitted)
	}
}

func TestMaxLenOverflowDropsRunes(t *testing.T) {
	e := NewSecureEntry(4)
	for _, r := range "abcdef" {
		e.TypedRune(r)
	}
	if got := e.Bytes(); !bytes.Equal(got, []byte("abcd")) {
		t.Fatalf("content = %q, want \"abcd\"", got)
	}
	if e.Len() != 4 {
		t.Fatalf("Len() = %d, want 4", e.Len())
	}
	if e.RuneCount() != 4 {
		t.Fatalf("RuneCount() = %d, want 4", e.RuneCount())
	}
	expectedCap := 4 * utf8.UTFMax
	if e.maxRunes != 4 {
		t.Fatalf("max runes is %d, expected 4", e.maxRunes)
	}
	if cap(e.buffer) != expectedCap {
		t.Fatalf("buffer capacity changed to %d, want %d", cap(e.buffer), expectedCap)
	}
}

func TestMultibyteRunes(t *testing.T) {
	e := NewSecureEntry(32)
	e.TypedRune('€') // 3 bytes (U+20AC)
	if e.Len() != 3 {
		t.Fatalf("Len() = %d, want 3 for '€'", e.Len())
	}
	if e.RuneCount() != 1 {
		t.Fatalf("RuneCount() = %d, want 1", e.RuneCount())
	}
	if !bytes.Equal(e.Bytes(), []byte{0xE2, 0x82, 0xAC}) {
		t.Fatalf("bytes = %x, want [e2 82 ac]", e.Bytes())
	}

	// Backspace must remove the whole rune.
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if e.Len() != 0 || e.RuneCount() != 0 {
		t.Fatalf("Len/RuneCount after backspace = %d/%d, want 0/0", e.Len(), e.RuneCount())
	}

	// An invalid lone continuation byte in the buffer must never corrupt
	// rune decoding on backspace.
	e2 := NewSecureEntry(8)
	copy(e2.buffer, []byte{0xE2})
	e2.count = 1
	e2.runeCount = 1
	e2.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if e2.Len() != 0 {
		t.Fatalf("Len() = %d, want 0 after backspace on partial rune", e2.Len())
	}
}

func TestEraseZerosEntireBuffer(t *testing.T) {
	e := NewSecureEntry(16)
	for _, r := range "topsecret" {
		e.TypedRune(r)
	}
	e.Erase()
	for i, b := range e.buffer {
		if b != 0 {
			t.Fatalf("buffer[%d] = %d, not zeroed after Erase", i, b)
		}
	}
	if e.Len() != 0 || e.RuneCount() != 0 {
		t.Fatalf("Len/RuneCount after Erase = %d/%d, want 0/0", e.Len(), e.RuneCount())
	}
}

func TestShortcutsIgnored(t *testing.T) {
	e := NewSecureEntry(16)
	e.TypedRune('s')

	e.TypedShortcut(&fyne.ShortcutPaste{})
	e.TypedShortcut(&fyne.ShortcutCopy{})
	e.TypedShortcut(&fyne.ShortcutCut{})
	e.TypedShortcut(&fyne.ShortcutSelectAll{})
	e.TypedShortcut(&fyne.ShortcutUndo{})
	e.TypedShortcut(&fyne.ShortcutRedo{})

	if got := e.Bytes(); !bytes.Equal(got, []byte("s")) {
		t.Fatalf("content after shortcuts = %q, want \"s\"", got)
	}
}

func TestDisabledIgnoresInput(t *testing.T) {
	e := NewSecureEntry(16)
	e.Disable()
	e.TypedRune('a')
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if e.Len() != 0 {
		t.Fatalf("disabled entry accepted input, Len() = %d", e.Len())
	}
	e.Enable()
	e.TypedRune('a')
	if e.Len() != 1 {
		t.Fatalf("enabled entry rejected input, Len() = %d", e.Len())
	}
}

func TestInvalidRunesIgnored(t *testing.T) {
	e := NewSecureEntry(8)
	e.TypedRune(0)
	if e.Len() != 0 {
		t.Fatalf("null rune accepted, Len() = %d", e.Len())
	}
}

func TestMaskRenderingNeverShowsContent(t *testing.T) {
	e := NewSecureEntry(8)
	for _, r := range "secret" {
		e.TypedRune(r)
	}
	e.SetMaskRune('*')
	r := e.CreateRenderer()
	defer r.Destroy()

	r.Refresh()
	var visibleMask string
	for _, o := range r.Objects() {
		text, ok := o.(*canvas.Text)
		if !ok {
			continue
		}
		if strings.Contains(text.Text, "secret") {
			t.Fatalf("renderer text contains raw content: %q", text.Text)
		}
		if text.Visible() {
			visibleMask = text.Text
		}
	}
	if visibleMask != "******" {
		t.Fatalf("unexpected displayed mask text: %q, want \"******\"", visibleMask)
	}
}

func TestRendererSmoke(t *testing.T) {
	e := NewSecureEntry(8)
	r := e.CreateRenderer()
	defer r.Destroy()

	min := r.MinSize()
	if min.Width <= 0 || min.Height <= 0 {
		t.Fatalf("MinSize = %v, want positive", min)
	}
	r.Layout(fyne.NewSize(200, 40))
	e.TypedRune('x')
	r.Refresh()
	if len(r.Objects()) != 5 {
		t.Fatalf("Objects() count = %d, want 5", len(r.Objects()))
	}
}
