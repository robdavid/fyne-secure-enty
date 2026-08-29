package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	secureentry "github.com/robdavid/fyne-secure-entry"
)

func main() {
	a := app.New()
	w := a.NewWindow("SecureEntry Demo")

	entry := secureentry.NewSecureEntry(32)
	entry.SetPlaceHolder("Enter a password...")

	status := widget.NewLabel("")

	entry.OnChanged = func(b []byte) {
		status.SetText(fmt.Sprintf("holding %d bytes (%d chars)", entry.Len(), entry.RuneCount()))
	}

	process := widget.NewButton("Process (hex)", func() {
		b := entry.Bytes()
		// Copy the content to a disposable buffer so the entries can keep
		// reading while we operate, then wipe the copy when done.
		buf := make([]byte, len(b))
		copy(buf, b)
		entry.Erase()
		status.SetText(fmt.Sprintf("processed %d bytes: %x (input erased)", len(buf), buf))
		os.Stdout.Write(buf)
		clear(buf)
	})

	erase := widget.NewButton("Erase", func() {
		entry.Erase()
		status.SetText("buffer erased from memory")
	})

	w.SetContent(container.NewBorder(
		nil, nil, nil, container.NewHBox(process, erase),
		container.NewVBox(entry, status),
	))
	w.Resize(fyne.NewSize(420, 140))
	w.ShowAndRun()
}
