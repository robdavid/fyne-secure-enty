# fyne-secure-entry

A [Fyne](https://fyne.io) text entry widget designed for secure data such as
passwords.

`SecureEntry` stores the user's input **only** in a fixed-length, zeroable
`[]byte` buffer. The content is never converted to a Go `string` — Go strings
are immutable and cannot be erased, so they would leave the input retained in
memory until garbage collection. Everything the widget draws on screen is
derived from a single non-sensitive mask rune (default `•`) repeated
`RuneCount()` times, so the actual content never leaks into the display layer,
the renderer, or the theme measurement code either.

## Installation

```sh
go get fyne.io/fyne/v2
```

The widget requires a Fyne build environment (C toolchain plus the platform
graphics libraries, e.g. Wayland/X11 dev headers for GLFW on Linux).

## Quick start

```go
package main

import (
	"bytes"

	"fyne.io/fyne/v2/app"

	secureentry "github.com/robdavid/fyne-secure-entry"
)

func main() {
	a := app.New()
	w := a.NewWindow("Login")

	entry := secureentry.NewSecureEntry(32) // up to 32 characters
	entry.SetPlaceHolder("Password")

	entry.OnSubmitted = func(b []byte) {
		// Process the raw bytes here (e.g. hash against a stored verifier).
		// Do NOT convert b to a string.
		login(b)
		entry.Erase() // wipe the buffer from memory once done
	}

	w.SetContent(entry)
	w.ShowAndRun()
}

func login(pw []byte) {
	// Example: equality check without materialising a string.
	want := []byte("hunter2")
	if bytes.Equal(pw, want) {
		// authenticated
	}
}
```

## Usage

### Create

```go
entry := secureentry.NewSecureEntry(maxRunes int)
```

`maxRunes` is the maximum number of **runes** (characters) the entry will
accept, regardless of how many UTF-8 bytes each rune occupies. The backing
buffer is allocated once as `maxRunes * utf8.UTFMax` bytes and is never
reallocated.

### Read

```go
b := entry.Bytes()
```

`Bytes()` returns the current content as a `[]byte` slice backed by the
widget's internal buffer. Process it purely as a byte slice:

- Do **not** convert it to a `string`.
- The returned slice is valid only until the next input event or a call to
  `Erase()`; copy anything you must keep for later into a buffer you manage.
- If you do copy it, remember to `clear()` your copy when finished.

Also available:

```go
entry.Len()       // number of content bytes currently held
entry.RuneCount() // number of characters currently held (== displayed mask count)
```

### Callbacks

```go
entry.OnChanged   = func(b []byte) { /* fires on every keypress / backspace */ }
entry.OnSubmitted = func(b []byte) { /* fires on Enter/Return */ }
```

Both receive the same erasable `[]byte` as `Bytes()`.

### Erase

```go
entry.Erase()
```

`Erase()` zeroes **the entire backing buffer** (all `maxRunes*4` bytes) with
Go's builtin `clear`, resets the content length, and clears the display. Call it
after processing the input, or whenever the widget is no longer needed. It is
safe to call at any time.

> Note: fyne-secure-entry is a widget, so it lives as long as its container
> does. When you drop it, call `Erase()` first; there is no GC hook to wipe the
> buffer for you.

### Styling

```go
entry.SetPlaceHolder("Password") // shown while the entry is empty
entry.SetMaskRune('●')           // change the conceal glyph (default '•')
entry.Disable()                  // / entry.Enable()  - entry.Disabled()
```

The widget follows the active Fyne theme and shows the standard blinking
cursor while focused.

### Demo

```sh
go run ./cmd/demo
```

A window with a `SecureEntry(32)`: type, click **Process (hex)** to see the
bytes and have the input erased, or **Erase** to wipe at any time.

## Security model

Guaranteed:

- Content exists only inside the fixed `[]byte` buffer, which can be fully
  erased with `Erase()`.
- The buffer is allocated once; edits use in-buffer `copy` (memmove), never
  `append`, so no reallocated copies are left behind.
- Insertion encodes each rune into a stack array which is zeroed on every
  return path, then copies it into the buffer.
- Backspace zeroes the bytes it frees.
- Editing is append-only: there is no cursor navigation, selection, undo/redo
  or clipboard (`TypedShortcut` ignores paste/copy/cut/select-all/undo/redo),
  so no other copy of the content is created, exported or retained.
- The display shows only mask glyphs (and the placeholder), never content.

Known limitations (inherent to Fyne / the OS, outside the widget's control):

- The clipboard and OS IME hold their own buffers; this widget never calls
  them. Mobile IME composition state lives outside Go memory.
- The rune is delivered to `TypedRune` by the driver, so one transient copy can
  exist in driver code you don't control.
- `clear()` reliably zeroes the widget's single internal buffer, and Go's
  non-moving GC means those bytes stay zero until the page is released. This is
  a contract, not a mechanism: the widget can only scrub what it owns. If the
  app drops the widget (or a container holding it) without calling `Erase()`,
  or keeps an un-cleared copy of `Bytes()`, that content *can* be freed
  uncleared. Likewise, content that passes through layers the widget cannot
  reach — the input driver's rune frames, mobile IME composition, and
  OS/hardware state such as swap, core dumps or physical RAM — cannot be
  scrubbed from user space.