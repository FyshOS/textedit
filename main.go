//go:generate fyne bundle -o data.go img/Icon.png
//go:generate fyne bundle -o icons.go img/format

package main

import (
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/x/fyne/menu"
)

func main() {
	a := app.New()
	a.SetIcon(resourceIconPng)
	w := a.NewWindow("TextEdit")

	edit := &textEdit{window: w, changed: binding.NewBool()}
	edit.recents = menu.NewRecents("Recent items...", func(u fyne.URI) {
		r, err := storage.Reader(u)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		edit.load(r)
	})
	ui := edit.makeUI(w)
	w.SetContent(ui)
	open := fyne.NewMenuItem("Open", edit.open)
	open.Icon = theme.NewThemedResource(theme.FolderOpenIcon())
	save := fyne.NewMenuItem("Save", edit.save)
	save.Icon = theme.NewThemedResource(theme.DocumentSaveIcon())
	fileMenu := fyne.NewMenu("File",
		open,
		edit.recents.MenuItem(),
		fyne.NewMenuItemSeparator(),
		save)
	w.SetMainMenu(fyne.NewMainMenu(fileMenu, edit.formatMenu))

	edit.changed.AddListener(binding.NewDataListener(func() {
		title := "TextEdit"
		if edit.uri != nil {
			title += ": " + edit.uri.Name()
		}
		edited, _ := edit.changed.Get()
		if edited {
			title += " *"
		}

		w.SetTitle(title)
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

	edit.focus()
	w.Resize(fyne.NewSize(480, 360))
	w.ShowAndRun()
}
