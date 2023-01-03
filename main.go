//go:generate fyne bundle -o data.go img/Icon.png

package main

import (
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)

func main() {
	a := app.New()
	a.SetIcon(resourceIconPng)
	w := a.NewWindow("TextEdit")

	edit := &textEdit{window: w, changed: binding.NewBool()}
	ui := edit.makeUI()
	w.SetContent(ui)

	edit.changed.AddListener(binding.NewDataListener(func() {
		edited, _ := edit.changed.Get()
		if edited {
			w.SetTitle("TextEdit *")
		} else {
			w.SetTitle("TextEdit")
		}
	}))

	if len(os.Args) > 1 {
		file := storage.NewFileURI(os.Args[1])
		read, err := storage.Reader(file)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			err = edit.load(read)
			if err != nil {
				dialog.ShowError(err, w)
			}
		}
	}

	w.Canvas().Focus(edit.entry)
	w.Resize(fyne.NewSize(480, 360))
	w.ShowAndRun()
}
