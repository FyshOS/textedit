package main

import (
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)

func (e *textEdit) load(r fyne.URIReadCloser) error {
	data, err := io.ReadAll(r)
	_ = r.Close()

	if err == nil {
		e.uri = r.URI()
		e.entry.SetText(string(data))
		e.changed.Set(false)
	}
	return err
}

func (e *textEdit) open() {
	dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, e.window)
			return
		}
		if r == nil {
			return
		}

		err = e.load(r)
		if err != nil {
			dialog.ShowError(err, e.window)
		}
	}, e.window)
}

func (e *textEdit) save() {
	if e.uri != nil {
		w, err := storage.Writer(e.uri)
		if err != nil {
			dialog.ShowError(err, e.window)
			return
		}

		err = e.saveAs(w)
		if err != nil {
			dialog.ShowError(err, e.window)
		}

		return
	}

	dialog.ShowFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, e.window)
			return
		}
		if w == nil {
			return
		}

		err = e.saveAs(w)
		if err != nil {
			dialog.ShowError(err, e.window)
		}
	}, e.window)
}

func (e *textEdit) saveAs(w fyne.URIWriteCloser) error {
	_, err := w.Write([]byte(e.entry.Text))
	if err != nil {
		return err
	}

	_ = w.Close()
	e.uri = w.URI()

	e.changed.Set(false)
	return nil
}
