package main

import (
	"io"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)

const (
	plainExtension = ".txt"
	richExtension  = ".md"
)

// isMarkdown reports whether a file should be edited as rich text.
func isMarkdown(u fyne.URI) bool {
	switch strings.ToLower(u.Extension()) {
	case richExtension, ".markdown":
		return true
	}
	return false
}

// fileName returns the name to suggest when saving, using the extension that matches the current mode.
func (e *textEdit) fileName() string {
	ext := plainExtension
	if e.richMode {
		ext = richExtension
	}

	if e.uri == nil {
		return "Untitled" + ext
	}
	return strings.TrimSuffix(e.uri.Name(), e.uri.Extension()) + ext
}

func (e *textEdit) load(r fyne.URIReadCloser) error {
	data, err := io.ReadAll(r)
	_ = r.Close()

	if err == nil {
		e.uri = r.URI()
		e.rename = false
		e.setMode(isMarkdown(e.uri), string(data))
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
	if e.uri != nil && !e.rename {
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

	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
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
	d.SetFileName(e.fileName())
	if e.uri != nil {
		// a file that changed mode is saved beside the one it was opened from
		if parent, err := storage.Parent(e.uri); err == nil {
			if dir, err := storage.ListerForURI(parent); err == nil {
				d.SetLocation(dir)
			}
		}
	}
	d.Show()
}

func (e *textEdit) saveAs(w fyne.URIWriteCloser) error {
	_, err := w.Write([]byte(e.text()))
	if err != nil {
		return err
	}

	_ = w.Close()
	e.uri = w.URI()
	e.rename = false

	e.changed.Set(false)
	return nil
}
